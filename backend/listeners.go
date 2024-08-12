package backend

import (
	"Mars/auth"
	"Mars/icarus"
	"Mars/structs"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/umahmood/haversine"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var WS_SEND_DELAY = time.Duration(auth.GetFromConf("WS_SEND_DELAY").(float64)) * time.Millisecond
var PING_EXPIRY = time.Duration(auth.GetFromConf("PING_EXPIRY").(float64)) * time.Second
var KNOWN_ENTITIES = GetKnownEntities()

var activeEntities = make(map[uint64]context.Context)
var ourEntities = make(map[uint64]string)
var droneToServerMap = make(map[uint64]auth.Server)
var serverToDroneMap = make(map[auth.Server][]uint64)
var entitiesForInit = make(map[uint64]structs.EntityInit)
var clients = make(map[*websocket.Conn]bool)
var broadcast = make(chan Message)
var clientMutex = &sync.Mutex{}
var entityMutex = &sync.Mutex{}
var wsMutex = &sync.Mutex{}
var cmdsToSend []map[string]any
var pingMap = make(map[string]time.Time)
var pingIdMap = make(map[uint64]Ping)
var pingMutex = &sync.Mutex{}
var initMutex = &sync.Mutex{}
var initDone = make(chan bool)
var marsEntityCancelMap = make(map[uint64]*context.CancelFunc)

func calculate3DDestination(lat1, lon1, alt1, lat2, lon2, alt2, distFromTarget float64) (float64, float64, float64, float64) {
	//distToEnemy2D := haversine2D(float32(lat1), float32(lon1), float32(lat2), float32(lon2))
	_, distToEnemy2D := haversine.Distance(haversine.Coord{Lat: lat1, Lon: lon1}, haversine.Coord{Lat: lat2, Lon: lon2})
	distToEnemy3D := math.Sqrt(math.Pow(distToEnemy2D, 2) + math.Pow(alt2-alt1, 2))
	targetPointDistance := distToEnemy3D - distFromTarget - 0.5
	distRatio := targetPointDistance / distToEnemy3D
	fmt.Printf("Distance to Enemy: %f km (%f 2D), Target Point Distance: %f km, Distance Ratio: %f\n", distToEnemy3D, distToEnemy2D, targetPointDistance, distRatio)

	latd := (lat2 - lat1) * distRatio
	lond := (lon2 - lon1) * distRatio
	altd := (alt2 - alt1) * distRatio

	lat3 := lat1 + latd
	fmt.Printf("%f = %f - %f\n", lat3, lat1, latd)
	lon3 := lon1 + lond
	fmt.Printf("%f = %f - %f\n", lon3, lon1, lond)
	alt3 := alt1 + altd
	fmt.Printf("%f = %f - %f\n", alt3, alt1, altd)

	return lat3, lon3, alt3, distToEnemy3D
}

func haversine2D(lat1, lon1, lat2, lon2 float32) float64 {
	const earthRadius = 6371.0
	lat1Rad := lat1 * math.Pi / 180
	lon1Rad := lon1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	lon2Rad := lon2 * math.Pi / 180

	dlat := lat2Rad - lat1Rad
	dlon := lon2Rad - lon1Rad

	a := math.Pow(math.Sin(float64(dlat/2)), 2) + math.Cos(float64(lat1Rad))*math.Cos(float64(lat2Rad))*math.Pow(math.Sin(float64(dlon/2)), 2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	// Calculate the distance
	distance := earthRadius * c
	return distance
}

func GetServerFromID(id uint64) (auth.Server, error) {
	for server, drones := range serverToDroneMap {
		for _, drone := range drones {
			if drone == id {
				return server, nil
			}
		}
	}
	return auth.Server{}, errors.New(fmt.Sprintf("Could not find server for id %d", id))
}

func updateInitSlice(id uint64, attribute string, value any) {
	initMutex.Lock()
	defer initMutex.Unlock()
	_, ok := entitiesForInit[id]
	if !ok {
		fmt.Printf("[UIS | %d] - Couldn't find entity\n", id)
		return
	}
	/*if id == 1004 {
		fmt.Printf("[UIS | %d] - Updating entity\n", id)
		fmt.Printf("[UIS | %d] - %s = %v\n", id, attribute, value)
	}*/
	entitiesForInit[id] = structs.UpdateEntity(entitiesForInit[id], attribute, value)
}

func PingCleaner() {
	for {
		time.Sleep(PING_EXPIRY)
		pingMutex.Lock()
		now := time.Now()
		for uid, expiry := range pingMap {
			if now.After(expiry) {
				delete(pingMap, uid)
				item := make(map[string]any)
				item["command"] = "removePing"
				uidUint64, err := strconv.ParseUint(uid, 10, 64)
				if err != nil {
					fmt.Println("Error:", err)
					return
				}
				item["id"] = uidUint64
				fmt.Printf("[CLEANER] - Ping %s removed\n", uid)
				go addCmd(item)
			}
		}
		pingMutex.Unlock()
	}
}

func addPing(ping Ping) {
	if _, found := ourEntities[ping.Id]; found {
		return
	}
	pingMutex.Lock()
	pingIdMap[ping.Id] = ping
	item := make(map[string]any)
	if _, found := pingMap[fmt.Sprintf("%d", ping.Id)]; !found {
		item["command"] = "newPing"
		item["id"] = ping.Id
		item["alt"] = ping.Altitude
		item["class"] = ping.Role
		item["lat"] = ping.Latitude
		item["lon"] = ping.Longitude
	} else {
		item["command"] = "movePing"
		item["id"] = ping.Id
		item["alt"] = ping.Altitude
		item["class"] = ping.Role
		item["lat"] = ping.Latitude
		item["lon"] = ping.Longitude
	}
	go addCmd(item)
	pingMap[fmt.Sprintf("%d", ping.Id)] = time.Time.Add(time.Now(), PING_EXPIRY)
	pingMutex.Unlock()
}

type KnownEntity struct {
	Type   string  `json:"type"`
	Lat    float64 `json:"lat"`
	Lon    float64 `json:"lon"`
	Alt    float64 `json:"alt"`
	Nation string  `json:"nation"`
}

func GetKnownEntities() map[string]KnownEntity {
	result, err := os.ReadFile("conf/known_entities.json")
	if err != nil {
		fmt.Println("Error reading known_entities.json - ", err.Error())
		os.Exit(1)
	}
	var knownEntities map[string]KnownEntity
	err = json.Unmarshal(result, &knownEntities)
	if err != nil {
		fmt.Println("Error unmarshalling known_entities.json - ", err.Error())
		os.Exit(1)
	}
	return knownEntities
}

type Ping struct {
	Id        uint64
	Role      string
	Latitude  float32
	Longitude float32
	Altitude  float32
	Heading   float32
	Nation    string
}

type Message struct {
	Msg  []byte
	Conn *websocket.Conn
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func addCmd(cmd map[string]any) {
	if len(cmd) == 0 {
		return
	}
	wsMutex.Lock()
	cmdsToSend = append(cmdsToSend, cmd)
	wsMutex.Unlock()
}

func addEntity(IFF uint64) {
	newCtx := context.WithoutCancel(context.Background())
	entityMutex.Lock()
	activeEntities[IFF] = newCtx
	entityMutex.Unlock()
}

func removeEntity(IFF uint64) {
	entityMutex.Lock()
	initMutex.Lock()
	delete(activeEntities, IFF)
	delete(entitiesForInit, IFF)
	initMutex.Unlock()
	entityMutex.Unlock()
}

// Handle incoming WebSocket connections
func Handler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade:", err)
		return
	}
	defer func(conn *websocket.Conn) {
		err := conn.Close()
		if err != nil {
			fmt.Println("[HANDLER] - Error closing connection: ", err)
		}
	}(conn)

	clientMutex.Lock()
	clients[conn] = true
	clientMutex.Unlock()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Read:", err)
			clientMutex.Lock()
			delete(clients, conn)
			clientMutex.Unlock()
			break
		}
		log.Printf("recv: %s", msg)
		if string(msg) == "init" {
			go UpdateDronePos(conn)
		} else {
			jsonString := string(msg)
			var jsonMap map[string]any
			err := json.Unmarshal([]byte(jsonString), &jsonMap)
			if err != nil {
				fmt.Printf("[RECV] - Error unmarshalling json: %s\n", err)
				log.Println(err)
			}
			if jsonMap["cmd"].(string) == "move" {
				server, cmdErr := GetServerFromID(uint64(jsonMap["id"].(float64)))
				if cmdErr != nil {
					fmt.Printf("[RECV] - Error getting server from ID %d\n", jsonMap["id"].(uint64))
					continue
				}
				if float32(jsonMap["alt"].(float64)) <= 3 {
					// takeoff
				}
				idFloat, ok := jsonMap["id"].(float64)
				if !ok {
					http.Error(w, "Invalid type for id", http.StatusBadRequest)
					return
				}
				id := uint64(idFloat)

				latFloat, ok := jsonMap["lat"].(float64)
				if !ok {
					http.Error(w, "Invalid type for lat", http.StatusBadRequest)
					return
				}
				lat := float32(latFloat)

				lonFloat, ok := jsonMap["lon"].(float64)
				if !ok {
					http.Error(w, "Invalid type for lon", http.StatusBadRequest)
					return
				}
				lon := float32(lonFloat)

				altFloat, ok := jsonMap["alt"].(float64)
				if !ok {
					http.Error(w, "Invalid type for alt", http.StatusBadRequest)
					return
				}
				alt := float32(altFloat)

				cancel, found := marsEntityCancelMap[id]
				if found {
					fmt.Printf("Cancelling %d current task\n", id)
					cancelFunc := *cancel
					cancelFunc()
					delete(marsEntityCancelMap, id)
					time.Sleep(1 * time.Second)
				}

				if ourEntities[id] != "BCT" {
					go MoveEntity(server.Client, id, lat, lon, alt)
				} else {
					go MoveLandEntity(server.Client, id, lat, lon)
				}

			} else if jsonMap["cmd"].(string) == "releaseDrone" {
				server, cmdErr := GetServerFromID(uint64(jsonMap["id"].(float64)))
				if cmdErr != nil {
					fmt.Printf("[RECV] - Error getting server from ID %0.f\n", jsonMap["id"].(float64))
					continue
				}
				fmt.Printf("[RECV] - Releasing drone %d\n", uint64(jsonMap["id"].(float64)))

				idFloat, ok := jsonMap["id"].(float64)
				if !ok {
					http.Error(w, "Invalid type for id", http.StatusBadRequest)
					return
				}
				id := uint64(idFloat)

				baseIdFloat, ok := jsonMap["baseId"].(float64)
				if !ok {
					http.Error(w, "Invalid type for baseId", http.StatusBadRequest)
					return
				}
				baseId := uint64(baseIdFloat)

				hangarIdFloat, ok := jsonMap["hangarId"].(float64)
				if !ok {
					http.Error(w, "Invalid type for hangarId", http.StatusBadRequest)
					return
				}
				hangarId := uint64(hangarIdFloat)

				go ReloadRelease(server.Client, id, baseId, hangarId)
				fmt.Println(server)
			} else if jsonMap["cmd"].(string) == "rightClickEntity" {
				myIdFloat, ok := jsonMap["myId"].(float64)
				if !ok {
					http.Error(w, "Invalid type for myId", http.StatusBadRequest)
					return
				}
				myId := uint64(myIdFloat)
				println(myId)
				targetIdFloat, ok := jsonMap["targetId"].(float64)
				if !ok {
					http.Error(w, "Invalid type for targetId", http.StatusBadRequest)
					return
				}
				targetId := uint64(targetIdFloat)

				_, pingFound := pingMap[fmt.Sprintf("%d", targetId)]
				_, ourEntitiesFound := ourEntities[targetId]

				if pingFound {
					if ourEntities[myId] == "BCT" {
						cancel, found := marsEntityCancelMap[myId]
						if found {
							fmt.Printf("Cancelling %d current task\n", myId)
							cancelFunc := *cancel
							cancelFunc()
							delete(marsEntityCancelMap, myId)
						}
						fmt.Printf("Found %d in pings. It is a %s\n", targetId, pingIdMap[targetId].Role)
						fmt.Printf("\t%f:%f\n", pingIdMap[targetId].Latitude, pingIdMap[targetId].Longitude)
						fmt.Println("Starting track goroutine")
						server, cmdErr := GetServerFromID(uint64(jsonMap["myId"].(float64)))
						if cmdErr != nil {
							fmt.Printf("[RECV] - Error getting server from ID %d\n", jsonMap["id"].(uint64))
							continue
						}
						go func() {
							navStreamReq := &icarus.StreamNavStatusRequest{EntityId: myId}
							streamCtx, cancel := context.WithCancel(context.Background())
							defer cancel()
							status, err := server.Client.Stream_Nav_Status(streamCtx, navStreamReq)
							fmt.Println("Starting stream")
							if err != nil {
								//TODO investigate if droneCancel needs to be called
								return
							}
							fmt.Println("Let's go")
							done, cancelFunc := context.WithCancel(context.Background())
							marsEntityCancelMap[myId] = &cancelFunc
							for {
								select {
								case <-done.Done():
									fmt.Printf("Cancelling intercept for %d\n", myId)
									return
								default:
									recv, err := status.Recv()
									fmt.Println(recv)
									if err != nil {
										if err == io.EOF {
											for err == io.EOF {
												fmt.Printf("[RECV | %d] - Stream closed by server\n", myId)
												status, err = server.Client.Stream_Nav_Status(streamCtx, navStreamReq)
											}
											continue
										}

										fmt.Printf("[RECV | %d] - Stream closed by server\n", myId)
										status, err = server.Client.Stream_Nav_Status(streamCtx, navStreamReq)
										continue
									}
									if recv == nil {
										fmt.Printf("[RECV | %d] - Recv is nil: %v\n", myId, err)
										continue
									}
									pingMutex.Lock()
									ping := pingIdMap[targetId]
									pingMutex.Unlock()
									var distFromTarget float64
									if ping.Altitude == 0 && ourEntities[myId] == "BCT" {
										distFromTarget = 1.5
									} else if ping.Altitude == 0 {
										distFromTarget = 0.5
									} else {
										distFromTarget = 5
									}
									tgtLat, tgtLon, _, distToTgt := calculate3DDestination(float64(recv.GetStatus().GetLatitude()),
										float64(recv.GetStatus().GetLongitude()), 0,
										float64(ping.Latitude), float64(ping.Longitude), 0, distFromTarget)

									tgtLat = float64(ping.Latitude)
									tgtLon = float64(ping.Longitude)
									fmt.Printf("Me\t\t%f:%f\n", recv.GetStatus().Latitude, recv.GetStatus().Longitude)
									fmt.Printf("Them\t\t%f:%f\n", ping.Latitude, ping.Longitude)
									MoveLandEntity(server.Client, myId, float32(tgtLat), float32(tgtLon))
									if distToTgt < distFromTarget+0.5 {
										for {
											select {
											case <-done.Done():
												fmt.Printf("Cancelling intercept for %d\n", myId)
												return
											default:
												pingMutex.Lock()
												ping := pingIdMap[targetId]
												pingMutex.Unlock()
												distFromTarget = 1.5
												initMutex.Lock()
												tgtLat, tgtLon, _, _ := calculate3DDestination(float64(entitiesForInit[myId].Lat),
													float64(entitiesForInit[myId].Lng), 0,
													float64(ping.Latitude), float64(ping.Longitude), 0, distFromTarget-0.5)
												initMutex.Unlock()
												MoveLandEntity(server.Client, myId, float32(tgtLat), float32(tgtLon))
												resp, err := server.Client.Get_Payloads(context.Background(), &icarus.GetPayloadsRequest{EntityId: myId})
												if err != nil {
													fmt.Printf("[RECV | %d] - Get_Payloads error: %v\n", myId, err)
													return
												}

												var payloadID uint64
												var payloadName string
												if ping.Altitude == 0 && ourEntities[myId] == "BCT" {
													payloadName = "SeekerMissile"
												} else if ping.Altitude == 0 {
													payloadName = "ThermalLance"
												} else {
													payloadName = "AntimatterMissile"
												}
												for _, payload := range resp.PayloadStatus {
													if strings.Contains(payload.Name, payloadName) {
														payloadID = uint64(payload.PayloadId)
														_, err = server.Client.Set_Payload_Enabled(context.Background(), &icarus.SetPayloadEnabledRequest{
															EntityId:  myId,
															PayloadId: payload.PayloadId,
															Enabled:   true})
														if err != nil {
															fmt.Printf("[RECV | %d] - Set_Payload_Enabled error: %v\n", myId, err)
															return
														}
													}
												}

												args := []*icarus.PayloadArguments{
													{
														Name:  "target",
														Type:  icarus.PayloadArgType_PayloadArgType_UINT,
														Value: strconv.FormatUint(uint64(targetId), 10),
													},
												}
												cmd := &icarus.PayloadCommand{
													Name:        "fire",
													Description: "Fire a missile at a specified aerial target",
													Args:        args,
												}

												req := &icarus.ExecutePayloadCommandRequest{
													EntityId:  myId,
													PayloadId: uint32(payloadID),
													Command:   cmd,
												}

												//Call Icarus to fire SAM at target id, currently defaults to payload id 0.
												_, err = server.Client.Execute_Payload_Command(context.Background(), req)
												if err != nil {
													fmt.Printf("[RECV | %d] - Execute_Payload_Command error: %v\n", myId, err)
													return
												}

												// Prints ouput of shooting if there are no errors
												fmt.Printf("[RECV | %d] - Execute_Payload_Command complete\n", myId)
												time.Sleep(60 * time.Second)
											}
										}
									}
									time.Sleep(1 * time.Second)
								}
							}
						}()
					} else {
						cancel, found := marsEntityCancelMap[myId]
						if found {
							fmt.Printf("Cancelling %d current task\n", myId)
							cancelFunc := *cancel
							cancelFunc()
							delete(marsEntityCancelMap, myId)
						}
						fmt.Printf("Found %d in pings. It is a %s\n", targetId, pingIdMap[targetId].Role)
						fmt.Printf("\t%f:%f\n", pingIdMap[targetId].Latitude, pingIdMap[targetId].Longitude)
						fmt.Println("Starting track goroutine")
						server, cmdErr := GetServerFromID(uint64(jsonMap["myId"].(float64)))
						if cmdErr != nil {
							fmt.Printf("[RECV] - Error getting server from ID %d\n", jsonMap["id"].(uint64))
							continue
						}
						go func() {
							navStreamReq := &icarus.StreamNavStatusRequest{EntityId: myId}
							streamCtx, cancel := context.WithCancel(context.Background())
							defer cancel()
							status, err := server.Client.Stream_Nav_Status(streamCtx, navStreamReq)
							fmt.Println("Starting stream")
							if err != nil {
								//TODO investigate if droneCancel needs to be called
								return
							}
							fmt.Println("Let's go")
							done, cancelFunc := context.WithCancel(context.Background())
							marsEntityCancelMap[myId] = &cancelFunc
							for {
								select {
								case <-done.Done():
									fmt.Printf("Cancelling intercept for %d\n", myId)
									return
								default:
									recv, err := status.Recv()
									fmt.Println(recv)
									if err != nil {
										if err == io.EOF {
											for err == io.EOF {
												fmt.Printf("[RECV | %d] - Stream closed by server\n", myId)
												status, err = server.Client.Stream_Nav_Status(streamCtx, navStreamReq)
											}
											continue
										}

										fmt.Printf("[RECV | %d] - Stream closed by server\n", myId)
										status, err = server.Client.Stream_Nav_Status(streamCtx, navStreamReq)
										continue
									}
									if recv == nil {
										fmt.Printf("[RECV | %d] - Recv is nil: %v\n", myId, err)
										continue
									}
									pingMutex.Lock()
									ping := pingIdMap[targetId]
									pingMutex.Unlock()
									var distFromTarget float64
									if ping.Altitude == 0 {
										distFromTarget = 1
										fmt.Println("Ground Target")
									} else {
										distFromTarget = 5.5
									}
									tgtLat, tgtLon, tgtAlt, distToTgt := calculate3DDestination(float64(recv.GetStatus().GetLatitude()),
										float64(recv.GetStatus().GetLongitude()), float64(recv.GetStatus().Altitude/1000),
										float64(ping.Latitude), float64(ping.Longitude), float64(ping.Altitude/1000), distFromTarget)
									if ping.Altitude == 0 {
										tgtAlt = 20
										tgtLat = float64(ping.Latitude)
										tgtLon = float64(ping.Longitude)
									}
									fmt.Printf("Me\t\t%f:%f\n", recv.GetStatus().Latitude, recv.GetStatus().Longitude)
									fmt.Printf("Them\t\t%f:%f\n", ping.Latitude, ping.Longitude)
									MoveEntity(server.Client, myId, float32(tgtLat), float32(tgtLon), float32(tgtAlt))
									if distToTgt < distFromTarget {
										resp, err := server.Client.Get_Payloads(context.Background(), &icarus.GetPayloadsRequest{EntityId: myId})
										if err != nil {
											fmt.Printf("[RECV | %d] - Get_Payloads error: %v\n", myId, err)
											return
										}

										var payloadID uint64
										var payloadName string
										if ping.Altitude == 0 {
											payloadName = "ThermalLance"
										} else {
											payloadName = "AntimatterMissile"
										}
										for _, payload := range resp.PayloadStatus {
											if strings.Contains(payload.Name, payloadName) {
												payloadID = uint64(payload.PayloadId)
												_, err = server.Client.Set_Payload_Enabled(context.Background(), &icarus.SetPayloadEnabledRequest{
													EntityId:  myId,
													PayloadId: payload.PayloadId,
													Enabled:   true})
												if err != nil {
													fmt.Printf("[RECV | %d] - Set_Payload_Enabled error: %v\n", myId, err)
													return
												}
											}
										}

										args := []*icarus.PayloadArguments{
											{
												Name:  "target",
												Type:  icarus.PayloadArgType_PayloadArgType_UINT,
												Value: strconv.FormatUint(uint64(targetId), 10),
											},
										}
										cmd := &icarus.PayloadCommand{
											Name:        "fire",
											Description: "Fire a missile at a specified aerial target",
											Args:        args,
										}

										req := &icarus.ExecutePayloadCommandRequest{
											EntityId:  myId,
											PayloadId: uint32(payloadID),
											Command:   cmd,
										}

										//Call Icarus to fire SAM at target id, currently defaults to payload id 0.
										_, err = server.Client.Execute_Payload_Command(context.Background(), req)
										if err != nil {
											fmt.Printf("[RECV | %d] - Execute_Payload_Command error: %v\n", myId, err)
											return
										}

										// Prints ouput of shooting if there are no errors
										fmt.Printf("[RECV | %d] - Execute_Payload_Command complete\n", myId)
										return
									}
									time.Sleep(1 * time.Second)
								}
							}
						}()
					}
				}
				if ourEntitiesFound {
					server, cmdErr := GetServerFromID(uint64(jsonMap["myId"].(float64)))
					if cmdErr != nil {
						fmt.Printf("[RECV] - Error getting server from ID %d\n", jsonMap["id"].(uint64))
						continue
					}
					cancel, found := marsEntityCancelMap[myId]
					if found {
						fmt.Printf("Cancelling %d current task\n", myId)
						cancelFunc := *cancel
						cancelFunc()
						delete(marsEntityCancelMap, myId)
					}
					if entitiesForInit[targetId].Class == "Airbase" {
						go MoveEntity(server.Client, myId, entitiesForInit[targetId].Lat,
							entitiesForInit[targetId].Lng, 10)
						done, cancelFunc := context.WithCancel(context.Background())
						marsEntityCancelMap[myId] = &cancelFunc
						go func() {
							fmt.Printf("[RECV | %d] - Starting RTB routine\n", myId)
							phase := 0

							targetLat := float32(entitiesForInit[targetId].Lat)
							targetLon := float32(entitiesForInit[targetId].Lng)
							navStreamReq := &icarus.StreamNavStatusRequest{EntityId: myId}
							streamCtx, cancel := context.WithCancel(context.Background())
							defer cancel()
							status, err := server.Client.Stream_Nav_Status(streamCtx, navStreamReq)
							if err != nil {
								//TODO investigate if droneCancel needs to be called
								return
							}
							for {
								recv, err := status.Recv()
								select {
								case <-done.Done():
									fmt.Printf("Cancelling RTB for %d\n", myId)
									return
								default:
									if err != nil {
										for err == io.EOF {
											fmt.Printf("[RECV | %d] - Stream closed by server\n", myId)
											status, err = server.Client.Stream_Nav_Status(streamCtx, navStreamReq)
											continue
										}

										fmt.Printf("[RECV | %d] - Stream closed by server\n", myId)
										status, err = server.Client.Stream_Nav_Status(streamCtx, navStreamReq)
										continue
									}
									if recv == nil {
										fmt.Printf("[RECV | %d] - Recv is nil: %v\n", myId, err)
										continue
									}
									curLat := recv.GetStatus().GetLatitude()
									curLon := recv.GetStatus().GetLongitude()
									curAlt := recv.GetStatus().GetAltitude()
									curVel := recv.GetStatus().GetVelocity()
									go updateInitSlice(myId, "lat", curLat)
									go updateInitSlice(myId, "lng", curLon)
									go updateInitSlice(myId, "alt", curAlt)
									switch phase {
									case 0:
										if haversine2D(curLat, curLon, targetLat, targetLon) < 0.05 && curVel < 3 {
											phase = 1
											go func() {
												landReq := &icarus.LandRequest{EntityId: myId}
												_, err = server.Client.Land(context.Background(), landReq)
												if err != nil {
													fmt.Printf("[RECV | %d] - Land error: %v\n", myId, err)
													phase = -1
													return
												}
											}()
										}
									case 1:
										if curAlt < 2 {
											phase = 2
										}
									case 2:
										phase = -1
										go LoadDrone(myId, targetId, server.Client)
									case -1:
										fmt.Printf("[RECV | %d] - Land complete, closing goroutine\n", myId)
										return
									}
								}
							}
						}()
					} else if entitiesForInit[targetId].Class == "Fort" && ourEntities[myId] == "BCT" {
						go MoveLandEntity(server.Client, myId, entitiesForInit[targetId].Lat,
							entitiesForInit[targetId].Lng)
						done, cancelFunc := context.WithCancel(context.Background())
						marsEntityCancelMap[myId] = &cancelFunc
						go func() {
							fmt.Printf("[RECV | %d] - Starting RTB routine\n", myId)
							phase := 0

							targetLat := float32(entitiesForInit[targetId].Lat)
							targetLon := float32(entitiesForInit[targetId].Lng)
							navStreamReq := &icarus.StreamNavStatusRequest{EntityId: myId}
							streamCtx, cancel := context.WithCancel(context.Background())
							defer cancel()
							status, err := server.Client.Stream_Nav_Status(streamCtx, navStreamReq)
							if err != nil {
								//TODO investigate if droneCancel needs to be called
								return
							}
							for {
								recv, err := status.Recv()
								select {
								case <-done.Done():
									fmt.Printf("Cancelling RTB for %d\n", myId)
									return
								default:
									if err != nil {
										for err == io.EOF {
											fmt.Printf("[RECV | %d] - Stream closed by server\n", myId)
											status, err = server.Client.Stream_Nav_Status(streamCtx, navStreamReq)
											continue
										}

										fmt.Printf("[RECV | %d] - Stream closed by server\n", myId)
										status, err = server.Client.Stream_Nav_Status(streamCtx, navStreamReq)
										continue
									}
									if recv == nil {
										fmt.Printf("[RECV | %d] - Recv is nil: %v\n", myId, err)
										continue
									}
									curLat := recv.GetStatus().GetLatitude()
									curLon := recv.GetStatus().GetLongitude()
									curVel := recv.GetStatus().GetVelocity()
									go updateInitSlice(myId, "lat", curLat)
									go updateInitSlice(myId, "lng", curLon)
									switch phase {
									case 0:
										if haversine2D(curLat, curLon, targetLat, targetLon) < 0.05 && curVel < 3 {
											go func() {
												LoadDrone(myId, targetId, server.Client)
											}()
											phase = -1
										}
									case -1:
										fmt.Printf("[RECV | %d] - Land complete, closing goroutine\n", myId)
										return
									}
								}
							}
						}()
					} else {
						done, cancelFunc := context.WithCancel(context.Background())
						marsEntityCancelMap[myId] = &cancelFunc
						go func() {
							fmt.Printf("[RECV | %d] - Starting escort routine for %d\n", myId, targetId)

							navStreamReq := &icarus.StreamNavStatusRequest{EntityId: myId}
							streamCtx, cancel := context.WithCancel(context.Background())
							defer cancel()
							status, err := server.Client.Stream_Nav_Status(streamCtx, navStreamReq)
							if err != nil {
								//TODO investigate if droneCancel needs to be called
								return
							}
							for {
								recv, err := status.Recv()
								select {
								case <-done.Done():
									fmt.Printf("Cancelling escort for %d\n", myId)
									return
								default:
									if err != nil {
										if err == io.EOF {
											for err == io.EOF {
												fmt.Printf("[RECV | %d] - Stream closed by server\n", myId)
												status, err = server.Client.Stream_Nav_Status(streamCtx, navStreamReq)
											}
											continue
										}

										fmt.Printf("[RECV | %d] - Stream closed by server\n", myId)
										status, err = server.Client.Stream_Nav_Status(streamCtx, navStreamReq)
										continue
									}
									if recv == nil {
										fmt.Printf("[RECV | %d] - Recv is nil: %v\n", myId, err)
										continue
									}
									curLat := recv.GetStatus().GetLatitude()
									curLon := recv.GetStatus().GetLongitude()
									curAlt := recv.GetStatus().GetAltitude()

									go updateInitSlice(myId, "lat", curLat)
									go updateInitSlice(myId, "lng", curLon)
									go updateInitSlice(myId, "alt", curAlt)

									initMutex.Lock()
									escortLat := float64(entitiesForInit[targetId].Lat)
									escortLon := float64(entitiesForInit[targetId].Lng)
									escortAlt := float64(entitiesForInit[targetId].Alt)
									initMutex.Unlock()

									targetLat, targetLon, _, _ := calculate3DDestination(float64(curLat),
										float64(curLon), float64(curAlt/1000), escortLat, escortLon, (escortAlt / 1000), 1)

									go MoveEntity(server.Client, myId, float32(targetLat), float32(targetLon),
										float32(escortAlt))
									time.Sleep(1 * time.Second)
								}
							}
						}()
					}
				}
			} else if jsonMap["cmd"].(string) == "land" {
				server, cmdErr := GetServerFromID(uint64(jsonMap["myId"].(float64)))
				if cmdErr != nil {
					fmt.Printf("[RECV] - Error getting server from ID %0.f\n", jsonMap["id"].(float64))
					continue
				}

				idFloat, ok := jsonMap["myId"].(float64)
				if !ok {
					http.Error(w, "Invalid type for id", http.StatusBadRequest)
					return
				}
				id := uint64(idFloat)
				class := ourEntities[id]
				go Land(id, server.Client, class)
			} else if jsonMap["cmd"].(string) == "takeoff" {
				server, cmdErr := GetServerFromID(uint64(jsonMap["myId"].(float64)))
				if cmdErr != nil {
					fmt.Printf("[RECV] - Error getting server from ID %0.f\n", jsonMap["id"].(float64))
					continue
				}

				idFloat, ok := jsonMap["myId"].(float64)
				if !ok {
					http.Error(w, "Invalid type for id", http.StatusBadRequest)
					return
				}
				id := uint64(idFloat)

				go takeoff(id, server.Client)
			} else if jsonMap["cmd"].(string) == "fire" {
				server, cmdErr := GetServerFromID(uint64(jsonMap["myId"].(float64)))
				if cmdErr != nil {
					fmt.Printf("[RECV] - Error getting server from ID %0.f\n", jsonMap["id"].(float64))
					continue
				}

				idFloat, ok := jsonMap["myId"].(float64)
				if !ok {
					http.Error(w, "Invalid type for id", http.StatusBadRequest)
					return
				}
				id := uint64(idFloat)

				targetFloat, ok := jsonMap["targetId"].(float64)
				if !ok {
					http.Error(w, "Invalid type for id", http.StatusBadRequest)
					return
				}
				targetID := uint64(targetFloat)

				airFloat, ok := jsonMap["air"].(float64)
				if !ok {
					http.Error(w, "Invalid type for id", http.StatusBadRequest)
					return
				}
				air := uint16(airFloat)

				if air == 0 {
					fireBomb(id, targetID, server.Client)
				} else {
					fireMissile(id, targetID, server.Client)
				}
			}
			fmt.Println(jsonMap)
		}
	}
}

func SendMessage() {
	ticker := time.NewTicker(WS_SEND_DELAY)
	defer ticker.Stop()

	for range ticker.C {
		clientMutex.Lock()
		wsMutex.Lock()
		if len(cmdsToSend) > 0 {
			jsonVer, _ := json.Marshal(cmdsToSend)
			broadcast <- Message{Msg: (jsonVer), Conn: nil}
			cmdsToSend = cmdsToSend[:0]
		}
		wsMutex.Unlock()
		clientMutex.Unlock()
	}
}

func SendInitMessage(msg []byte, conn *websocket.Conn) {
	broadcast <- Message{Msg: msg, Conn: conn}
}

// Handle sending messages
func handleMessages() {
	for {
		msg := <-broadcast
		clientMutex.Lock()
		for client := range clients {
			if msg.Conn == nil || client == msg.Conn {
				err := client.WriteMessage(websocket.TextMessage, msg.Msg)
				if err != nil {
					log.Printf("Write: %v", err)
					err := client.Close()
					if err != nil {
						return
					}
					delete(clients, client)
				}
			}
		}
		clientMutex.Unlock()
	}
}

func init() {
	go auth.GetServerDetails()
	fmt.Println("[INIT] - Waiting for server initialisation")
	<-auth.ServerInitDone
	fmt.Println("[INIT] - Server initialised")
	go handleMessages()
	go SendMessage()
	setupClients := 0
	for index, server := range auth.Servers {
		if server.Client == nil {
			fmt.Printf("[INIT | %s] - Init failed, client is nil\n", server.IP)
			continue
		}
		fmt.Println(index, server)
		reqStatus := &icarus.GetAllNavStatusRequest{}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		dronesStatus, err := server.Client.Get_All_Nav_Status(ctx, reqStatus)
		if err != nil {
			fmt.Printf("[INIT | %d] - Error initialising Icarus Client for stream threads: %v\n", index, err)
			continue
		}
		var tempEntitiesSlice []uint64
		for _, entity := range dronesStatus.GetStatus() {
			if entity.GetStatus().GetHealth() == 0 {
				fmt.Printf("[INIT] - Entity %d dead\n", entity.GetEntityId())
				continue
			}
			tempEntitiesSlice = append(tempEntitiesSlice, entity.GetEntityId())
			if !entity.GetStatus().Carried {
				reqPayload := &icarus.GetPayloadsRequest{EntityId: entity.GetEntityId()}
				entityPayload, payloadErr := server.Client.Get_Payloads(ctx, reqPayload)
				for entityPayload == nil || payloadErr != nil {
					entityPayload, payloadErr = server.Client.Get_Payloads(ctx, reqPayload)
					fmt.Printf("[INIT] - Error getting payloads for %d\n", entity.GetEntityId())
				}

				payloads := make(map[string]any)
				for _, payload := range entityPayload.GetPayloadStatus() {
					payloads[fmt.Sprintf("%d", payload.PayloadId)] = payload
				}
				entitiesForInit[entity.EntityId] = structs.NewEntity(
					entity.GetStatus().Role,
					entity.GetStatus().GetLatitude(),
					entity.GetStatus().GetLongitude(),
					entity.GetStatus().GetAltitude(),
					entity.GetEntityId(),
					entity.GetStatus().Name,
					server.Nation,
					entity.GetStatus().Health,
					payloads)
			}
			fmt.Printf("[INIT] - Added %d to our entities\n", entity.GetEntityId())
			ourEntities[entity.EntityId] = entity.GetStatus().Role
			newCtx, newCancel := context.WithCancel(context.Background())
			_, ok := activeEntities[entity.GetEntityId()]
			if !entity.GetStatus().Carried && !ok {
				activeEntities[entity.GetEntityId()] = newCtx
			}
			go monitorPayloadChanges(entity.GetEntityId(), server.Client, newCtx, newCancel)
			go monitorNavChanges(entity.GetEntityId(), server, newCtx, newCancel)
			reqSensor := &icarus.GetSensorsRequest{EntityId: entity.GetEntityId()}
			sensors, err := server.Client.Get_Sensors(newCtx, reqSensor)
			for sensors == nil || err != nil {
				sensors, err = server.Client.Get_Sensors(newCtx, reqSensor)
				fmt.Printf("[INIT] - Error getting sensors for %d\n", entity.GetEntityId())
			}
			if sensors.Sensors != nil {
				go monitorSensorChanges(entity.GetEntityId(), server.Client, newCtx, newCancel)
			}
		}
		fmt.Printf("[INIT | %s] - Init finished\n", server.IP)
		serverToDroneMap[server] = tempEntitiesSlice
		setupClients++
	}
	if setupClients == 0 {
		fmt.Printf("[INIT] - failed to setup any Icarus clients\n")
		os.Exit(1)
	}
	close(initDone)
	go PingCleaner()
}

func UpdateDronePos(conn *websocket.Conn) {
	var cmds []map[string]any
	for _, entity := range entitiesForInit {
		item := make(map[string]any)
		item["command"] = "add"
		item["lat"] = entity.Lat
		item["lng"] = entity.Lng
		item["alt"] = entity.Alt
		item["class"] = entity.Class
		item["id"] = entity.Id
		item["name"] = entity.Name
		item["nation"] = entity.Nation
		item["health"] = entity.Health
		item["payloads"] = entity.Payloads
		cmds = append(cmds, item)
	}
	jsonVer, err := json.Marshal(cmds)
	if err != nil {
		fmt.Printf("[UDP | %v] - Error marshalling commands: %v\n", conn.RemoteAddr(), err)
		return
	}
	SendInitMessage(jsonVer, conn)
}

func monitorSensorChanges(entityID uint64, icarusClient icarus.IcarusClient, droneCtx context.Context,
	droneCancel context.CancelFunc) {
	<-initDone
	sensorStreamReq := &icarus.StreamAllSensorDataRequest{EntityId: entityID}

	streamCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	status, err := icarusClient.Stream_All_Sensor_Data(streamCtx, sensorStreamReq)
	if err != nil {
		//TODO investigate if droneCancel needs to be called
		return
	}

	for {
		select {
		case <-droneCtx.Done():
			fmt.Printf("[SENSOR | %d] - Stream killed by context\n", entityID)
			return
		default:
			recv, err := status.Recv()
			for err != nil {
				fmt.Printf("[SENSOR | %d] - Stream closed by server - %v\n", entityID, err.Error())
				status, err = icarusClient.Stream_All_Sensor_Data(streamCtx, sensorStreamReq)
			}
			if recv == nil {
				fmt.Printf("[SENSOR | %d] - Recv is nil: %v\n", entityID, err)
				continue
			}

			//fmt.Printf("[SENSOR | %d] - %v\n", entityID, recv)
			ping := recv.Ping

			newPing := Ping{
				Id:        ping.Id,
				Role:      ping.Role,
				Latitude:  ping.Latitude,
				Longitude: ping.Longitude,
				Altitude:  ping.Altitude,
				Heading:   ping.Heading,
				Nation:    "Unknown",
			}

			if entity, found := KNOWN_ENTITIES[fmt.Sprintf("%d", ping.Id)]; found {
				newPing.Nation = entity.Nation
			}

			go addPing(newPing)
		}
	}
}

func monitorPayloadChanges(entityID uint64, icarusClient icarus.IcarusClient, droneCtx context.Context,
	droneCancel context.CancelFunc) {
	<-initDone
	payloadStreamReq := &icarus.StreamAllPayloadsRequest{EntityId: entityID}

	streamCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	status, err := icarusClient.Stream_All_Payloads(streamCtx, payloadStreamReq)
	if err != nil {
		// TODO investigate if droneCancel needs to be called
		return
	}

	for {
		select {
		case <-droneCtx.Done():
			fmt.Printf("[PAYLOAD | %d] - Stream killed by context\n", entityID)
			return
		default:
			recv, err := status.Recv()
			for err != nil {
				log.Printf("[PAYLOAD | %d] - Failed to receive from stream: %v", entityID, err)
				status, err = icarusClient.Stream_All_Payloads(streamCtx, payloadStreamReq)
			}
			if recv == nil {
				fmt.Printf("[PAYLOAD | %d] - Recv is nil: %v\n", entityID, err)
				continue
			}

			//fmt.Println("Payload - ", recv)
			item := make(map[string]any)
			item["command"] = "updatePayload"
			item["id"] = entityID
			item["payload_id"] = recv.PayloadId
			item["new_quantity"] = recv.CurrentQuantity
			if recv.CarriedEntities != nil {
				item["carried_entities"] = recv.CarriedEntities
			}
			go updateInitSlice(entityID, "payloads", []any{recv.PayloadId, recv.CurrentQuantity})
			go addCmd(item)
		}
	}
}

func monitorNavChanges(entityID uint64, server auth.Server, droneCtx context.Context,
	droneCancel context.CancelFunc) {
	<-initDone
	icarusClient := server.Client
	navStreamReq := &icarus.StreamNavStatusRequest{EntityId: entityID}

	streamCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	status, err := icarusClient.Stream_Nav_Status(streamCtx, navStreamReq)
	if err != nil {
		//TODO investigate if droneCancel needs to be called
		return
	}
	for {
		select {
		case <-droneCtx.Done():
			fmt.Printf("[NAV | %d] - Stream killed by context\n", entityID)
			return
		default:
			for status == nil {
				fmt.Printf("[NAV | %d] - Stream closed by server\n", entityID)
				status, err = icarusClient.Stream_Nav_Status(streamCtx, navStreamReq)
			}
			recv, err := status.Recv()
			if err != nil {
				if errors.Is(err, context.Canceled) {
					fmt.Printf("[NAV | %d] - Stream killed by context\n", entityID)
					return
				}
				if err == io.EOF {
					for err == io.EOF {
						fmt.Printf("[NAV | %d] - Stream closed by server\n", entityID)
						status, err = icarusClient.Stream_Nav_Status(streamCtx, navStreamReq)
					}
					continue
				}

				fmt.Printf("[NAV | %d] - Stream closed by server\n", entityID)
				status, err = icarusClient.Stream_Nav_Status(streamCtx, navStreamReq)
				continue
			}
			if recv == nil {
				fmt.Printf("[NAV | %d] - Recv is nil: %v\n", entityID, err)
				continue
			}
			//fmt.Printf("[NAV | %d] - %v\n", entityID, recv)
			item := make(map[string]any)
			_, ok := activeEntities[recv.EntityId]
			if !ok {
				entityReq := &icarus.GetEntityRequest{EntityId: entityID}
				entityResp, err := icarusClient.Get_Entity(streamCtx, entityReq)
				if err != nil {
					if errors.Is(err, context.Canceled) || strings.Contains(err.Error(), "context canceled") {
						fmt.Printf("[NAV | %d] - Stream killed by context\n", entityID)
						return
					}
					log.Printf("[NAV | %d] - Failed to receive entity data: %v\n", entityID, err)
					continue
				}
				if !entityResp.GetEntity().GetNavData().GetCarried() {
					entity := entityResp.Entity
					item["command"] = "add"
					item["class"] = entity.GetNavData().Role
					item["callsign"] = entity.GetNavData().Name
					item["lat"] = entity.GetNavData().GetLatitude()
					item["lng"] = entity.GetNavData().GetLongitude()
					item["id"] = entity.GetId()
					item["name"] = entity.GetNavData().Name
					item["nation"] = server.Nation
					item["alt"] = entity.GetNavData().Altitude
					item["health"] = entity.GetNavData().Health
					item["vel"] = entity.GetNavData().Velocity
					payloads := make(map[string]any)
					for _, payload := range entity.GetPayloads() {
						payloads[fmt.Sprintf("%d", payload.PayloadId)] = payload
						fmt.Println(payload)
						fmt.Printf("%s - %d/%d\n", payload.Name, payload.CurrentQuantity, payload.MaxQuantity)
					}
					item["payloads"] = payloads
					go addEntity(entityID)

					entitiesForInit[entity.Id] = structs.NewEntity(
						entity.NavData.Role,
						entity.NavData.GetLatitude(),
						entity.NavData.GetLongitude(),
						entity.NavData.GetAltitude(),
						entity.Id,
						entity.NavData.Name,
						server.Nation,
						entity.NavData.Health,
						payloads)
				}
			} else {
				if recv.GetStatus().GetHealth() <= 0 {
					item["command"] = "removeEntity"
					item["id"] = recv.GetEntityId()
					go addCmd(item)
					removeEntity(entityID)
					fmt.Printf("[NAV | %d] - Drone killed\n", entityID)
					droneCancel()
				}
				if recv.GetStatus().Altitude > 2 || ourEntities[entityID] == "BCT" {
					item["command"] = "updateNav"
					item["lat"] = recv.GetStatus().GetLatitude()
					item["lng"] = recv.GetStatus().GetLongitude()
					item["id"] = recv.GetEntityId()
					item["alt"] = recv.GetStatus().Altitude
					item["vel"] = recv.GetStatus().GetVelocity()
					item["health"] = recv.GetStatus().Health
					item["heading"] = recv.GetStatus().GetHeading()
					go updateInitSlice(entityID, "lat", recv.GetStatus().GetLatitude())
					go updateInitSlice(entityID, "lng", recv.GetStatus().GetLongitude())
					go updateInitSlice(entityID, "alt", recv.GetStatus().GetAltitude())
				} else {
					time.Sleep(5 * time.Second)
					statusReq := &icarus.GetEntityRequest{EntityId: entityID}
					statusResp, err := icarusClient.Get_Entity(droneCtx, statusReq)
					for err != nil || statusResp == nil {
						log.Printf("[NAV | %d] - Failed to receive entity nav status for %d: %v\n", entityID, entityID, err)
						statusResp, err = icarusClient.Get_Entity(droneCtx, statusReq)
						time.Sleep(10 * time.Second)
					}
					if statusResp.GetEntity().GetNavData().Carried {
						item["command"] = "removeEntity"
						item["id"] = recv.GetEntityId()
						removeEntity(entityID)
					}
				}
			}
			go addCmd(item)
		}
	}
}
