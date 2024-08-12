package backend

import (
	"Mars/icarus"
	"context"
	"fmt"
	"github.com/umahmood/haversine"
	"strconv"
	"strings"
	"time"
)

func MoveEntity(client icarus.IcarusClient, id uint64, lat float32, lon float32, alt float32) {
	if alt < 20 {
		alt = 20
	} else if alt > 4000 {
		alt = 4000
	}
	moveReq := &icarus.GoToRequest{
		EntityId: id,
		Waypoint: &icarus.Waypoint{
			Latitude:  lat,
			Longitude: lon,
			Altitude:  alt,
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, err := client.Go_To(ctx, moveReq)
	if err != nil {
		fmt.Printf("[MCMD | %d] - Failed to move entity: %s\n", id, err.Error())
		return
	}
	fmt.Printf("[MCMD | %d] - Moving entity to %f:%f at %.0fm\n", id, lat, lon, alt)
	//fmt.Printf("[MCMD | %d] - Moved entity to: %s\n", id, to)
}

func MoveLandEntity(client icarus.IcarusClient, id uint64, lat float32, lon float32) {
	moveReq := &icarus.GoToRequest{
		EntityId: id,
		Waypoint: &icarus.Waypoint{
			Latitude:  lat,
			Longitude: lon,
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, err := client.Go_To(ctx, moveReq)
	if err != nil {
		fmt.Printf("[MCMD | %d] - Failed to move entity: %s\n", id, err.Error())
		return
	}
	fmt.Printf("[MCMD | %d] - Moving entity to %f:%f\n", id, lat, lon)
}

func ReloadRelease(client icarus.IcarusClient, id uint64, baseId uint64, hangarId uint64) {
	payloadNameMap := make(map[string]string)
	payloadNameMap["TLLoader"] = "ThermalLance"
	payloadNameMap["AMMLoader"] = "AntimatterMissile"
	payloadNameMap["SeekerMissileLoader"] = "SeekerMissile"
	payloadNameMap["FuelLoader"] = "Fuel"
	dronePayloads, err := client.Get_Payloads(context.Background(), &icarus.GetPayloadsRequest{EntityId: id})
	if err != nil {
		fmt.Printf("[MCMD | %d] - Failed to get payloads: %s\n", id, err.Error())
		return
	}
	airbasePayloads, err := client.Get_Payloads(context.Background(), &icarus.GetPayloadsRequest{EntityId: baseId})
	if err != nil {
		fmt.Printf("[MCMD | %d] - Failed to get airbase payloads: %s\n", id, err.Error())
		return
	}
	fmt.Println(airbasePayloads.GetPayloadStatus())
	for payloadId, status := range dronePayloads.GetPayloadStatus() {
		currentQuantity := status.CurrentQuantity
		maxQuantity := status.MaxQuantity
		if currentQuantity == maxQuantity {
			continue
		}
		toFill := maxQuantity - currentQuantity
		var loaderId uint32 = 999
		for _, loaderStatus := range airbasePayloads.GetPayloadStatus() {
			if payloadNameMap[loaderStatus.Name] == status.Name {
				fmt.Printf("Matched %s to %s: %d\n", status.Name, loaderStatus.Name, status.PayloadId)
				loaderId = loaderStatus.PayloadId
				break
			}
		}
		if loaderId == 999 {
			fmt.Printf("[MCMD | %d] - Failed to find airbase payload id for %s\n", id, status.Name)
			continue
		}
		reloadReq := &icarus.ExecutePayloadCommandRequest{
			EntityId:  baseId,
			PayloadId: loaderId,
			Command: &icarus.PayloadCommand{
				Name: "load",
				Args: []*icarus.PayloadArguments{
					{
						//Target id
						Value: fmt.Sprint(id),
					},
					{
						//Quantity
						Value: fmt.Sprint(toFill),
					},
					{
						//Target payload id
						Value: fmt.Sprint(payloadId),
					},
				},
			},
		}
		//fmt.Println(fmt.Sprint(payloadId))
		//fmt.Println("Paylod id: ", payloadId, " - ", reloadReq)
		fmt.Println(reloadReq)
		_, err := client.Execute_Payload_Command(context.Background(), reloadReq)
		if err != nil {
			fmt.Printf("[MCMD | %d] - Failed to reload %s: %s\n", id, status.Name, err.Error())
			return
		}
	}
	releaseReq := &icarus.ExecutePayloadCommandRequest{
		EntityId:  baseId,
		PayloadId: uint32(hangarId),
		Command: &icarus.PayloadCommand{
			Name: "unload",
			Args: []*icarus.PayloadArguments{
				{
					Value: fmt.Sprintf("%d", id),
				},
			},
		},
	}

	// Execute the command to unload the drone from the hangar
	_, err = client.Execute_Payload_Command(context.Background(), releaseReq)
	if err != nil {
		fmt.Printf("[MCMD | %d] - Failed to release: %s\n", id, err.Error())
		return
	}

	takeoffReq := &icarus.TakeoffRequest{
		EntityId:        id,
		TakeoffAltitude: float32(50),
	}

	// Pass the context and the request to the Takeoff method on the IcarusClient
	_, err = client.Takeoff(context.Background(), takeoffReq)
	if err != nil {
		fmt.Printf("[MCMD | %d] - Failed to takeoff: %s\n", id, err.Error())
		return
	}
}

func LoadDrone(id, baseId uint64, client icarus.IcarusClient) {
	//This function will attempt to load or unload a drone from a hangar based on its current status

	// Retrieve the entity from Icarus and extract the class name from its NavData
	entityResponse, err := client.Get_Entity(context.Background(), &icarus.GetEntityRequest{
		EntityId: id})
	if err != nil {
		fmt.Printf("[MCMD | %d] - Failed to get entity: %s\n", id, err.Error())
		return
	}
	droneClass := entityResponse.Entity.GetNavData().GetClassName()

	// Loop through all payloads in the base and find the hangar that goes with the drone class
	var hangarId uint32
	basePayloads, _ := client.Get_Payloads(context.Background(), &icarus.GetPayloadsRequest{
		EntityId: baseId,
	})
	for _, payloadStatus := range basePayloads.GetPayloadStatus() {
		if strings.Contains(payloadStatus.GetName(), droneClass) {
			hangarId = payloadStatus.GetPayloadId()
		}
	}

	// Create an instance of the ExecutePayloadCommandRequest struct and populate it with the necessary data
	req := &icarus.ExecutePayloadCommandRequest{
		EntityId:  baseId,
		PayloadId: hangarId,
		Command: &icarus.PayloadCommand{
			Name: "load",
			Args: []*icarus.PayloadArguments{
				{
					//Load ID
					Value: fmt.Sprintf("%d", id),
				},
			},
		},
	}

	_, err = client.Execute_Payload_Command(context.Background(), req)
	if err != nil {
		fmt.Printf("[MCMD | %d] - Failed to load: %s\n", id, err.Error())
		return
	}
}

func Land(id uint64, client icarus.IcarusClient, class string) {
	landReq := &icarus.LandRequest{EntityId: id}
	_, err := client.Land(context.Background(), landReq)
	if err != nil {
		fmt.Printf("[RECV | %d] - Land error: %v\n", id, err)
		return
	}
	if class == "BCT" {
		go lastStand(id, client)
	}
}

func takeoff(id uint64, client icarus.IcarusClient) {
	takeoffReq := &icarus.TakeoffRequest{
		EntityId:        id,
		TakeoffAltitude: float32(50),
	}

	// Pass the context and the request to the Takeoff method on the IcarusClient
	_, err := client.Takeoff(context.Background(), takeoffReq)
	if err != nil {
		fmt.Printf("[MCMD | %d] - Failed to takeoff: %s\n", id, err.Error())
		return
	}
}

func fireMissile(id, target uint64, client icarus.IcarusClient) {
	resp, err := client.Get_Payloads(context.Background(), &icarus.GetPayloadsRequest{EntityId: id})
	if err != nil {
		fmt.Printf("[RECV | %d] - Get_Payloads error: %v\n", id, err)
		return
	}

	var payloadID uint64
	for _, payload := range resp.PayloadStatus {
		if strings.Contains(payload.Name, "AntimatterMissile") {
			payloadID = uint64(payload.PayloadId)
			_, err = client.Set_Payload_Enabled(context.Background(), &icarus.SetPayloadEnabledRequest{
				EntityId:  id,
				PayloadId: payload.PayloadId,
				Enabled:   true})
			if err != nil {
				fmt.Printf("[RECV | %d] - Set_Payload_Enabled error: %v\n", id, err)
				return
			}
		}
	}

	args := []*icarus.PayloadArguments{
		{
			Name:  "target",
			Type:  icarus.PayloadArgType_PayloadArgType_UINT,
			Value: strconv.FormatUint(uint64(target), 10),
		},
	}
	cmd := &icarus.PayloadCommand{
		Name:        "fire",
		Description: "Fire a missile at a specified aerial target",
		Args:        args,
	}

	req := &icarus.ExecutePayloadCommandRequest{
		EntityId:  id,
		PayloadId: uint32(payloadID),
		Command:   cmd,
	}

	//Call Icarus to fire SAM at target id, currently defaults to payload id 0.
	_, err = client.Execute_Payload_Command(context.Background(), req)
	if err != nil {
		fmt.Printf("[RECV | %d] - Execute_Payload_Command error: %v\n", id, err)
		return
	}

	// Prints output of shooting if there are no errors
	fmt.Printf("[RECV | %d] - Execute_Payload_Command complete\n", id)
	return
}

func fireBomb(id, target uint64, client icarus.IcarusClient) {
	resp, err := client.Get_Payloads(context.Background(), &icarus.GetPayloadsRequest{EntityId: id})
	if err != nil {
		fmt.Printf("[RECV | %d] - Get_Payloads error: %v\n", id, err)
		return
	}

	var payloadID uint64
	for _, payload := range resp.PayloadStatus {
		if strings.Contains(payload.Name, "ThermalLance") {
			payloadID = uint64(payload.PayloadId)
			_, err = client.Set_Payload_Enabled(context.Background(), &icarus.SetPayloadEnabledRequest{
				EntityId:  id,
				PayloadId: payload.PayloadId,
				Enabled:   true})
			if err != nil {
				fmt.Printf("[RECV | %d] - Set_Payload_Enabled error: %v\n", id, err)
				return
			}
		}
	}

	args := []*icarus.PayloadArguments{
		{
			Name:  "target",
			Type:  icarus.PayloadArgType_PayloadArgType_UINT,
			Value: strconv.FormatUint(uint64(target), 10),
		},
	}
	cmd := &icarus.PayloadCommand{
		Name:        "fire",
		Description: "Fire a bomb at a specified ground target",
		Args:        args,
	}

	req := &icarus.ExecutePayloadCommandRequest{
		EntityId:  id,
		PayloadId: uint32(payloadID),
		Command:   cmd,
	}

	//Call Icarus to fire SAM at target id, currently defaults to payload id 0.
	_, err = client.Execute_Payload_Command(context.Background(), req)
	if err != nil {
		fmt.Printf("[RECV | %d] - Execute_Payload_Command error: %v\n", id, err)
		return
	}

	// Prints output of shooting if there are no errors
	fmt.Printf("[RECV | %d] - Execute_Payload_Command complete\n", id)
	return
}

func lastStand(id uint64, client icarus.IcarusClient) {
	fmt.Printf("\n\n\tLast Stand\n\n")
	time.Sleep(3 * time.Second)
	locReq := &icarus.GetEntityRequest{EntityId: id}
	reqCtx := context.Background()
	entity, entityErr := client.Get_Entity(reqCtx, locReq)
	for entityErr != nil {
		fmt.Printf("\t[RECV | %d] - Get_Entity error: %v\n", id, entityErr)
		time.Sleep(1 * time.Second)
		entity, entityErr = client.Get_Entity(reqCtx, locReq)
	}

	entityLat := entity.GetEntity().GetNavData().GetLatitude()
	entityLon := entity.GetEntity().GetNavData().GetLongitude()

	sensorStreamReq := &icarus.StreamAllSensorDataRequest{EntityId: 1018}

	streamCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	status, err := client.Stream_All_Sensor_Data(streamCtx, sensorStreamReq)
	if err != nil {
		//TODO investigate if droneCancel needs to be called
		return
	}
	for {
		recv, err := status.Recv()
		for err != nil {
			fmt.Printf("[SENSOR | %d] - Stream closed by server - %v\n", id, err.Error())
			status, err = client.Stream_All_Sensor_Data(streamCtx, sensorStreamReq)
		}
		if recv == nil {
			fmt.Printf("[SENSOR | %d] - Recv is nil: %v\n", id, err)
			continue
		}

		//fmt.Printf("[SENSOR | %d] - %v\n", entityID, recv)
		ping := recv.Ping
		_, kmDist := haversine.Distance(haversine.Coord{Lat: float64(entityLat), Lon: float64(entityLon)},
			haversine.Coord{Lat: float64(ping.Latitude), Lon: float64(ping.Longitude)})
		for kmDist < 2 {
			var payloadID uint64
			payloadName := "SeekerMissile"

			for _, payload := range entity.GetEntity().Payloads {
				if strings.Contains(payload.Name, payloadName) {
					payloadID = uint64(payload.PayloadId)
					_, err = client.Set_Payload_Enabled(context.Background(), &icarus.SetPayloadEnabledRequest{
						EntityId:  id,
						PayloadId: payload.PayloadId,
						Enabled:   true})
					if err != nil {
						fmt.Printf("[RECV | %d] - Set_Payload_Enabled error: %v\n", id, err)
						return
					}
				}
			}

			args := []*icarus.PayloadArguments{
				{
					Name:  "target",
					Type:  icarus.PayloadArgType_PayloadArgType_UINT,
					Value: strconv.FormatUint(uint64(ping.Id), 10),
				},
			}
			cmd := &icarus.PayloadCommand{
				Name:        "fire",
				Description: "Fire a missile at a specified aerial target",
				Args:        args,
			}

			req := &icarus.ExecutePayloadCommandRequest{
				EntityId:  id,
				PayloadId: uint32(payloadID),
				Command:   cmd,
			}

			//Call Icarus to fire SAM at target id, currently defaults to payload id 0.
			_, err = client.Execute_Payload_Command(context.Background(), req)
			if err != nil {
				fmt.Printf("[RECV | %d] - Execute_Payload_Command error: %v\n", id, err)
				return
			}

			// Prints output of shooting if there are no errors
			fmt.Printf("[RECV | %d] - Execute_Payload_Command complete\n", id)
			time.Sleep(60 * time.Second)
		}
	}
}

func contains(s []uint64, searchTerm uint64) bool {
	for _, value := range s {
		if value == searchTerm {
			return true
		}
	}
	return false
}
