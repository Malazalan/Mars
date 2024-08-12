const { AirBase , Drone, Entity, KnownEntity, entityTypes, genIcon, Ping } = await import("./classes/entity.js");
let { allEntity, allPing, allStaticEntity, showBases, showDrones,
    showPings } = await import("./classes/entity.js");
import {getSidebarEntity, setSidebarEntity} from "./classes/entity.js";

const { addToMap, removeFromMap } = await import("./map.js");

let wsUrl, wsUrlVal;
const knownEntitiesData = await loadKnownEntities()

async function loadConfig() {
    const response = await fetch('../../conf/conf.json');
    if (!response.ok) {
        throw new Error('Failed to fetch conf.json: ' + response.statusText);
    }
    return await response.json();
}

async function loadKnownEntities() {
    const response = await fetch ('../../conf/known_entities.json');
    if (!response.ok) {
        throw new Error('Failed to fetch known_entities.json: ' + response.statusText);
    }
    const data = await response.json();
    return data;
}

async function init() {
    try {
        const confData = await loadConfig();
        wsUrl = 'ws://' + confData.Mars_IP + ':' + confData.Mars_PORT + '/ws';
        //wsUrlVal = 'ws://10.76.83.205:3001/ws'
        console.log(wsUrl);

        console.log(knownEntitiesData)
        for (const [id, data] of Object.entries(    knownEntitiesData)) {
            console.log(`${id} has ${data}`)
            for (const [key, value] of Object.entries(data)) {
                console.log(`  ${key} has ${value}`)
            }
            if (data.lat && data.lon) {
                const newEntityIcon = genIcon(data.type, data.nation || "unknown",
                    data.name || data.id || "unknown")
                const newEntity = new KnownEntity(
                    data.lat,
                    data.lon,
                    newEntityIcon,
                    id || "unknown",
                    data.name || "unknown",
                    data.type,
                    data.nation || "unknown",
                    true,
                    null
                )
                allStaticEntity[data.id] = newEntity
                addToMap(newEntity)
                newEntity.addEventListener("click", function() {
                    showEntityInfo(newEntity)
                })
            }
        }
    } catch (error) {
        console.error('Error loading configuration or modules:', error);
    }
}

await init();

console.log(wsUrl);
export const socket = new WebSocket(wsUrl);
socket.onopen = function () {
    console.log('WebSocket Connected!');
    socket.send("init")
}

export const socketVal = new WebSocket("ws://10.76.83.205:3001/ws");
socketVal.onopen = function () {
    console.log("Connection opened");
    socketVal.send("init");
    socketVal.addEventListener('message', function (event) {
        //console.log('Message from server: ', event.data);
        const jsonData = JSON.parse(event.data);
        console.log(jsonData)
        for (let i = 0; i < jsonData.length; i++) {
            console.log(`${i}/${jsonData.length} - ${jsonData[i].command}`)
            if (jsonData[i] === undefined) {
                continue
            }
            if (jsonData[i].command === "updateNav") {
                if (jsonData[i].health === 0 || !jsonData[i].health) {
                    let idToRm = allEntity[jsonData[i].id]
                    removeFromMap(allEntity[jsonData[i].id])
                    delete allEntity.idToRm
                    continue
                }
                console.log(allEntity[jsonData[i].id])
                allEntity[jsonData[i].id].setLatLng(L.latLng(jsonData[i].lat, jsonData[i].lng));
                allEntity[jsonData[i].id].alt = jsonData[i].alt;
                allEntity[jsonData[i].id].health = jsonData[i].health;
                const sidebarEntity = Number(getSidebarEntity())
                const id = Number(jsonData[i].id)
                if (sidebarEntity === id) {
                    let healthDiv = document.getElementById("health")
                    healthDiv.innerHTML = "Health: " + allEntity[jsonData[i].id].health || 0

                    let payloadDiv = document.getElementById("posDiv")
                    posDiv.innerHTML = "LatLng: " + allEntity[jsonData[i].id].getLatLng().lat + "," + allEntity[jsonData[i].id].getLatLng().lng

                    let altDiv = document.getElementById("alt")
                    altDiv.innerHTML = "Alt: " + (jsonData[i].alt || 0)

                    let velDiv = document.getElementById("vel")
                    velDiv.innerHTML = "Vel: " + (jsonData[i].vel || 0)
                }
            } else if (jsonData[i].command === "updatePayload") {
                console.log("Payload recv - ", jsonData)
                let entity = allEntity[jsonData[i].id]
                entity.setRotation(jsonData[i].heading)
                console.log("[", entity.name, "] Updating ", entity.payloads[jsonData[i].payload_id], " to ", jsonData[i].new_quantity)
                entity.payloads[jsonData[i].payload_id].current_quantity = jsonData[i].new_quantity
                if (entity instanceof AirBase) {
                    entity.payloads[jsonData[i].payload_id].carried_entities = jsonData[i].carried_entities
                }
                const sidebarEntity = Number(getSidebarEntity())
                const id = Number(jsonData[i].id)
                if (sidebarEntity === id) {
                    let payloadDiv = document.getElementById(jsonData[i].payload_id)
                    payloadDiv.innerHTML = entity.payloads[jsonData[i].payload_id].name + ": " +
                        entity.payloads[jsonData[i].payload_id].current_quantity + "/" +
                        entity.payloads[jsonData[i].payload_id].max_quantity;
                    if (entity.payloads[jsonData[i].payload_id].name.toString().includes("Hangar")) {
                        var hangarDiv = document.createElement('div');
                        hangarDiv.style.marginLeft = "1vh"
                        const hangar = entity.payloads[jsonData[i].payload_id].carried_entities
                        for (let drone in hangar) {
                            var droneDivWrapper = document.createElement("div");
                            var droneDiv = document.createElement("h3");
                            var releaseButton = document.createElement("button")

                            droneDivWrapper.style.margin = "0"

                            droneDiv.style.margin = "0"
                            droneDiv.innerHTML = "ID: " + hangar[drone]

                            releaseButton.innerHTML = "Release"
                            releaseButton.addEventListener("click", function() {
                                const message = JSON.stringify({
                                    cmd: 'releaseDrone',
                                    id: hangar[drone],
                                    baseId: entity.id,
                                    hangarId: Number(payload_index)
                                });
                                sendMessage(message)
                            })

                            droneDivWrapper.appendChild(droneDiv)
                            droneDivWrapper.appendChild(releaseButton)
                            hangarDiv.appendChild(droneDivWrapper)
                        }
                        payloadDiv.appendChild(hangarDiv);
                        payloadDiv.appendChild(hangarDiv);
                    }

                }
            } else if (jsonData[i].command === "add") {
                console.log(`Adding ${jsonData[i].id}`)
                if (jsonData[i].class === "Multi" ||
                    jsonData[i].class === "Fighter" ||
                    jsonData[i].class === "Bomber" ||
                    jsonData[i].class === "ISR" ||
                    jsonData[i].class === "Transport" ||
                    jsonData[i].class === "BCT") {
                    const newEntityIcon = genIcon(jsonData[i].class, jsonData[i].nation, jsonData[i].name)
                    const newEntity = new Drone(
                        jsonData[i].lat,
                        jsonData[i].lng,
                        newEntityIcon,
                        jsonData[i].id,
                        jsonData[i].name,
                        jsonData[i].class,
                        jsonData[i].nation,
                        true,
                        jsonData[i].alt || 0,
                        jsonData[i].payloads,
                        jsonData[i].vel || 0,
                        jsonData[i].health || 0
                    )
                    //newEntity.bindPopup(newEntity.alt)
                    console.log("Binding: ", newEntity.name, " - ", newEntity, " - ", newEntity.alt)
                    //allEntity[jsonData[i].id] = newEntity
                    addToMap(newEntity)
                    newEntity.addEventListener("click", function() {
                        showEntityInfo(newEntity)
                    })
                    /*newEntity.addEventListener("contextmenu", function() {
                        const myEntity = allEntity[getSidebarEntity()]
                        const message = JSON.stringify({
                            cmd: 'rightClickEntity',
                            targetId: newEntity.id,
                            myId: myEntity.id
                        });
                        sendMessage(message)
                    })
                    if (!showDrones) {
                        newEntity.setOpacity(0)
                    }*/
                } else if (jsonData[i].class === "Airbase") {
                    const newEntityIcon = genIcon(jsonData[i].class, jsonData[i].nation, jsonData[i].name)
                    const newEntity = new AirBase(
                        jsonData[i].lat,
                        jsonData[i].lng,
                        newEntityIcon,
                        jsonData[i].id,
                        jsonData[i].name,
                        jsonData[i].class,
                        jsonData[i].nation,
                        true,
                        jsonData[i].payloads,
                        jsonData[i].health || 0
                    )
                    //newEntity.bindPopup(newEntity.name)
                    //allEntity[jsonData[i].id] = newEntity
                    addToMap(newEntity)
                    newEntity.addEventListener("click", function () {
                        showEntityInfo(newEntity)
                    })
                    /*newEntity.addEventListener("contextmenu", function() {
                        const myEntity = allEntity[getSidebarEntity()]
                        const message = JSON.stringify({
                            cmd: 'rightClickEntity',
                            targetId: newEntity.id,
                            myId: myEntity.id
                        });
                        sendMessage(message)
                    })*/
                    if (!showBases) {
                        newEntity.setOpacity(0)
                    }
                } else {
                    const newEntityIcon = genIcon(jsonData[i].class, jsonData[i].nation, jsonData[i].name)
                    const newEntity = new Entity(
                        jsonData[i].lat,
                        jsonData[i].lng,
                        newEntityIcon,
                        jsonData[i].id,
                        jsonData[i].name,
                        jsonData[i].class,
                        jsonData[i].nation,
                        true,
                        jsonData[i].payloads,
                        jsonData[i].health || 0
                    )
                    //newEntity.bindPopup(newEntity.name)
                    //allEntity[jsonData[i].id] = newEntity
                    addToMap(newEntity)
                    newEntity.addEventListener("click", function () {
                        showEntityInfo(newEntity)
                    })
                    /*newEntity.addEventListener("contextmenu", function() {
                        const myEntity = allEntity[getSidebarEntity()]
                        const message = JSON.stringify({
                            cmd: 'rightClickEntity',
                            targetId: newEntity.id,
                            myId: myEntity.id
                        });
                        sendMessage(message)
                    })*/
                    if (!showBases) {
                        newEntity.setOpacity(0)
                    }
                }
            } else if (jsonData[i].command === "removeEntity") {
                let idToRm = allEntity[jsonData[i].id]
                removeFromMap(allEntity[jsonData[i].id])
                delete allEntity.idToRm
            } else if (jsonData[i].command === "newPing") {
                console.log("newPing - ", jsonData[i])
                const newEntityIcon = genIcon(jsonData[i].class, jsonData[i].nation, jsonData[i].id || "unknown")
                let lat = parseFloat(jsonData[i].lat);
                let lon = parseFloat(jsonData[i].lon);

                if (!isNaN(lat) && !isNaN(lon)) {
                    let nation = jsonData[i].nation
                    if (jsonData[i].class === "Airliner") {
                        nation = "Civilian"
                    }
                    var newPing = new Ping(
                        lat,
                        lon,
                        newEntityIcon,
                        jsonData[i].id,
                        jsonData[i].role,
                        nation
                    );

                    newPing.setOpacity(1)

                    if (!newPing || !newPing.getLatLng()) {
                        console.error(`New ping is null or has invalid latLng - ${JSON.stringify(jsonData[i])} - ${newPing}`)
                    } else {
                        allPing[jsonData[i].id] = newPing;
                        if (!(jsonData[i].id in knownEntitiesData)) {
                            addToMap(newPing);
                        }
                        newPing.addEventListener("click", function() {
                            if (getSidebarEntity()) {
                                console.warn(newPing.id)
                            } else {
                                console.error("No drone controlled")
                            }
                        })
                        newPing.addEventListener("contextmenu", function() {
                            const myEntity = allEntity[getSidebarEntity()]
                            const message = JSON.stringify({
                                cmd: 'rightClickEntity',
                                targetId: jsonData[i].id,
                                myId: myEntity.id
                            });
                            sendMessage(message)
                        })
                    }
                } else {
                    console.error(`Invalid lat/lon values: ${lat}, ${lon}`);
                }
            } else if (jsonData[i].command === "movePing") {
                console.log("movePing - ", jsonData[i])
                console.warn(jsonData[i].id in allPing)
                if (!(jsonData[i].id in allPing)) {
                    const newEntityIcon = genIcon(jsonData[i].class, jsonData[i].nation, jsonData[i].id || "unknown")
                    let lat = parseFloat(jsonData[i].lat);
                    let lon = parseFloat(jsonData[i].lon);
                    console.log(!isNaN(lat))
                    console.log(!isNaN(lon))
                    if (!isNaN(lat) && !isNaN(lon)) {
                        console.log("poo")
                        let nation = jsonData[i].nation
                        if (jsonData[i].class === "Airliner") {
                            nation = "Civilian"
                        }
                        var newPing = new Ping(
                            lat,
                            lon,
                            newEntityIcon,
                            jsonData[i].id,
                            jsonData[i].role,
                            nation
                        );

                        if (!newPing || !newPing.getLatLng()) {
                            console.error(`New ping is null or has invalid latLng - ${JSON.stringify(jsonData[i])} - ${newPing}`)
                        } else {
                            allPing[jsonData[i].id] = newPing;
                            if (!(jsonData[i].id in knownEntitiesData)) {
                                addToMap(newPing);
                            }
                            newPing.addEventListener("click", function() {
                                if (getSidebarEntity()) {
                                    console.warn(newPing.id)
                                } else {
                                    console.error("No drone controlled")
                                }
                            })
                            newPing.addEventListener("contextmenu", function() {
                                const myEntity = allEntity[getSidebarEntity()]
                                const message = JSON.stringify({
                                    cmd: 'rightClickEntity',
                                    targetId: jsonData[i].id,
                                    myId: myEntity.id
                                });
                                sendMessage(message)
                            })
                        }
                    } else {
                        console.error(`Invalid lat/lon values: ${lat}, ${lon}`);
                    }
                } else {
                    console.log(`Moving to ${jsonData[i].lat}:${jsonData[i].lon}`)
                    allPing[jsonData[i].id].setLatLng(L.latLng(jsonData[i].lat, jsonData[i].lon));
                    /*if (!showPings) {
                        newPing.setOpacity(0)
                    }*/
                    allPing[jsonData[i].id].setOpacity(1)
                }
                if (jsonData[i].id in allStaticEntity) {
                    allStaticEntity[jsonData[i].id].confirmed = true
                }
            } else if (jsonData[i].command === "removePing") {
                let pingToRemove = allPing[jsonData[i].id]
                delete allPing[jsonData[i].id]
                removeFromMap(pingToRemove)
                console.log("removePing - ", jsonData[i])
            }
        }
    })
};

socketVal.onerror = function (error) {
    console.error("WebSocket error observed:", error);
    // Handle the error (e.g., retry logic, show a message to the user)
};

socket.addEventListener('message', function (event) {
    //console.log('Message from server: ', event.data);
    const jsonData = JSON.parse(event.data);
    console.log(jsonData)
    for (let i = 0; i < jsonData.length; i++) {
        console.log(`${i}/${jsonData.length} - ${jsonData[i].command}`)
        if (jsonData[i] === undefined) {
            continue
        }
        if (jsonData[i].command === "updateNav") {
            if (jsonData[i].health === 0 || !jsonData[i].health) {
                let idToRm = allEntity[jsonData[i].id]
                removeFromMap(allEntity[jsonData[i].id])
                delete allEntity.idToRm
                continue
            }
            console.log(allEntity[jsonData[i].id])
            allEntity[jsonData[i].id].setLatLng(L.latLng(jsonData[i].lat, jsonData[i].lng));
            allEntity[jsonData[i].id].alt = jsonData[i].alt;
            allEntity[jsonData[i].id].health = jsonData[i].health;
            const sidebarEntity = Number(getSidebarEntity())
            const id = Number(jsonData[i].id)
            if (sidebarEntity === id) {
                let healthDiv = document.getElementById("health")
                healthDiv.innerHTML = "Health: " + allEntity[jsonData[i].id].health || 0

                let payloadDiv = document.getElementById("posDiv")
                posDiv.innerHTML = "LatLng: " + allEntity[jsonData[i].id].getLatLng().lat + "," + allEntity[jsonData[i].id].getLatLng().lng

                let altDiv = document.getElementById("alt")
                altDiv.innerHTML = "Alt: " + (jsonData[i].alt || 0)

                let velDiv = document.getElementById("vel")
                velDiv.innerHTML = "Vel: " + (jsonData[i].vel || 0)
            }
        } else if (jsonData[i].command === "updatePayload") {
            console.log("Payload recv - ", jsonData)
            let entity = allEntity[jsonData[i].id]
            entity.setRotation(jsonData[i].heading)
            console.log("[", entity.name, "] Updating ", entity.payloads[jsonData[i].payload_id], " to ", jsonData[i].new_quantity)
            entity.payloads[jsonData[i].payload_id].current_quantity = jsonData[i].new_quantity
            if (entity instanceof AirBase) {
                entity.payloads[jsonData[i].payload_id].carried_entities = jsonData[i].carried_entities
            }
            const sidebarEntity = Number(getSidebarEntity())
            const id = Number(jsonData[i].id)
            if (sidebarEntity === id) {
                let payloadDiv = document.getElementById(jsonData[i].payload_id)
                payloadDiv.innerHTML = entity.payloads[jsonData[i].payload_id].name + ": " +
                    entity.payloads[jsonData[i].payload_id].current_quantity + "/" +
                    entity.payloads[jsonData[i].payload_id].max_quantity;
                if (entity.payloads[jsonData[i].payload_id].name.toString().includes("Hangar")) {
                    var hangarDiv = document.createElement('div');
                    hangarDiv.style.marginLeft = "1vh"
                    const hangar = entity.payloads[jsonData[i].payload_id].carried_entities
                    for (let drone in hangar) {
                        var droneDivWrapper = document.createElement("div");
                        var droneDiv = document.createElement("h3");
                        var releaseButton = document.createElement("button")

                        droneDivWrapper.style.margin = "0"

                        droneDiv.style.margin = "0"
                        droneDiv.innerHTML = "ID: " + hangar[drone]

                        releaseButton.innerHTML = "Release"
                        releaseButton.addEventListener("click", function() {
                            const message = JSON.stringify({
                                cmd: 'releaseDrone',
                                id: hangar[drone],
                                baseId: entity.id,
                                hangarId: Number(payload_index)
                            });
                            sendMessage(message)
                        })

                        droneDivWrapper.appendChild(droneDiv)
                        droneDivWrapper.appendChild(releaseButton)
                        hangarDiv.appendChild(droneDivWrapper)
                    }
                    payloadDiv.appendChild(hangarDiv);
                    payloadDiv.appendChild(hangarDiv);
                }

            }
        } else if (jsonData[i].command === "add") {
            console.log(`Adding ${jsonData[i].id}`)
            if (jsonData[i].class === "Multi" ||
                jsonData[i].class === "Fighter" ||
                jsonData[i].class === "Bomber" ||
                jsonData[i].class === "ISR" ||
                jsonData[i].class === "Transport" ||
                jsonData[i].class === "BCT") {
                const newEntityIcon = genIcon(jsonData[i].class, jsonData[i].nation, jsonData[i].name)
                const newEntity = new Drone(
                    jsonData[i].lat,
                    jsonData[i].lng,
                    newEntityIcon,
                    jsonData[i].id,
                    jsonData[i].name,
                    jsonData[i].class,
                    jsonData[i].nation,
                    true,
                    jsonData[i].alt || 0,
                    jsonData[i].payloads,
                    jsonData[i].vel || 0,
                    jsonData[i].health || 0
                )
                //newEntity.bindPopup(newEntity.alt)
                console.log("Binding: ", newEntity.name, " - ", newEntity, " - ", newEntity.alt)
                allEntity[jsonData[i].id] = newEntity
                addToMap(newEntity)
                newEntity.addEventListener("click", function() {
                    showEntityInfo(newEntity)
                })
                newEntity.addEventListener("contextmenu", function() {
                    const myEntity = allEntity[getSidebarEntity()]
                    const message = JSON.stringify({
                        cmd: 'rightClickEntity',
                        targetId: newEntity.id,
                        myId: myEntity.id
                    });
                    sendMessage(message)
                })
                if (!showDrones) {
                    newEntity.setOpacity(0)
                }
            } else if (jsonData[i].class === "Airbase") {
                const newEntityIcon = genIcon(jsonData[i].class, jsonData[i].nation, jsonData[i].name)
                const newEntity = new AirBase(
                    jsonData[i].lat,
                    jsonData[i].lng,
                    newEntityIcon,
                    jsonData[i].id,
                    jsonData[i].name,
                    jsonData[i].class,
                    jsonData[i].nation,
                    true,
                    jsonData[i].payloads,
                    jsonData[i].health || 0
                )
                //newEntity.bindPopup(newEntity.name)
                allEntity[jsonData[i].id] = newEntity
                addToMap(newEntity)
                newEntity.addEventListener("click", function () {
                    showEntityInfo(newEntity)
                })
                newEntity.addEventListener("contextmenu", function() {
                    const myEntity = allEntity[getSidebarEntity()]
                    const message = JSON.stringify({
                        cmd: 'rightClickEntity',
                        targetId: newEntity.id,
                        myId: myEntity.id
                    });
                    sendMessage(message)
                })
                if (!showBases) {
                    newEntity.setOpacity(0)
                }
            } else {
                const newEntityIcon = genIcon(jsonData[i].class, jsonData[i].nation, jsonData[i].name)
                const newEntity = new Entity(
                    jsonData[i].lat,
                    jsonData[i].lng,
                    newEntityIcon,
                    jsonData[i].id,
                    jsonData[i].name,
                    jsonData[i].class,
                    jsonData[i].nation,
                    true,
                    jsonData[i].payloads,
                    jsonData[i].health || 0
                )
                //newEntity.bindPopup(newEntity.name)
                allEntity[jsonData[i].id] = newEntity
                addToMap(newEntity)
                newEntity.addEventListener("click", function () {
                    showEntityInfo(newEntity)
                })
                newEntity.addEventListener("contextmenu", function() {
                    const myEntity = allEntity[getSidebarEntity()]
                    const message = JSON.stringify({
                        cmd: 'rightClickEntity',
                        targetId: newEntity.id,
                        myId: myEntity.id
                    });
                    sendMessage(message)
                })
                if (!showBases) {
                    newEntity.setOpacity(0)
                }
            }
        } else if (jsonData[i].command === "removeEntity") {
            let idToRm = allEntity[jsonData[i].id]
            removeFromMap(allEntity[jsonData[i].id])
            delete allEntity.idToRm
        } else if (jsonData[i].command === "newPing") {
            console.log("newPing - ", jsonData[i])
            const newEntityIcon = genIcon(jsonData[i].class, jsonData[i].nation, jsonData[i].id || "unknown")
            let lat = parseFloat(jsonData[i].lat);
            let lon = parseFloat(jsonData[i].lon);

            if (!isNaN(lat) && !isNaN(lon)) {
                let nation = jsonData[i].nation
                if (jsonData[i].class === "Airliner") {
                    nation = "Civilian"
                }
                var newPing = new Ping(
                    lat,
                    lon,
                    newEntityIcon,
                    jsonData[i].id,
                    jsonData[i].role,
                    nation
                );

                newPing.setOpacity(1)

                if (!newPing || !newPing.getLatLng()) {
                    console.error(`New ping is null or has invalid latLng - ${JSON.stringify(jsonData[i])} - ${newPing}`)
                } else {
                    allPing[jsonData[i].id] = newPing;
                    if (!(jsonData[i].id in knownEntitiesData)) {
                        addToMap(newPing);
                    }
                    newPing.addEventListener("click", function() {
                        if (getSidebarEntity()) {
                            console.warn(newPing.id)
                        } else {
                            console.error("No drone controlled")
                        }
                    })
                    newPing.addEventListener("contextmenu", function() {
                        const myEntity = allEntity[getSidebarEntity()]
                        const message = JSON.stringify({
                            cmd: 'rightClickEntity',
                            targetId: jsonData[i].id,
                            myId: myEntity.id
                        });
                        sendMessage(message)
                    })
                }
            } else {
                console.error(`Invalid lat/lon values: ${lat}, ${lon}`);
            }
        } else if (jsonData[i].command === "movePing") {
            console.log("movePing - ", jsonData[i])
            console.warn(jsonData[i].id in allPing)
            if (!(jsonData[i].id in allPing)) {
                const newEntityIcon = genIcon(jsonData[i].class, jsonData[i].nation, jsonData[i].id || "unknown")
                let lat = parseFloat(jsonData[i].lat);
                let lon = parseFloat(jsonData[i].lon);
                console.log(!isNaN(lat))
                console.log(!isNaN(lon))
                if (!isNaN(lat) && !isNaN(lon)) {
                    console.log("poo")
                    let nation = jsonData[i].nation
                    if (jsonData[i].class === "Airliner") {
                        nation = "Civilian"
                    }
                    var newPing = new Ping(
                        lat,
                        lon,
                        newEntityIcon,
                        jsonData[i].id,
                        jsonData[i].role,
                        nation
                    );

                    if (!newPing || !newPing.getLatLng()) {
                        console.error(`New ping is null or has invalid latLng - ${JSON.stringify(jsonData[i])} - ${newPing}`)
                    } else {
                        allPing[jsonData[i].id] = newPing;
                        if (!(jsonData[i].id in knownEntitiesData)) {
                            addToMap(newPing);
                        }
                        newPing.addEventListener("click", function() {
                            if (getSidebarEntity()) {
                                console.warn(newPing.id)
                            } else {
                                console.error("No drone controlled")
                            }
                        })
                        newPing.addEventListener("contextmenu", function() {
                            const myEntity = allEntity[getSidebarEntity()]
                            const message = JSON.stringify({
                                cmd: 'rightClickEntity',
                                targetId: jsonData[i].id,
                                myId: myEntity.id
                            });
                            sendMessage(message)
                        })
                    }
                } else {
                    console.error(`Invalid lat/lon values: ${lat}, ${lon}`);
                }
            } else {
                console.log(`Moving to ${jsonData[i].lat}:${jsonData[i].lon}`)
                allPing[jsonData[i].id].setLatLng(L.latLng(jsonData[i].lat, jsonData[i].lon));
                /*if (!showPings) {
                    newPing.setOpacity(0)
                }*/
                allPing[jsonData[i].id].setOpacity(1)
            }
            if (jsonData[i].id in allStaticEntity) {
                allStaticEntity[jsonData[i].id].confirmed = true
            }
        } else if (jsonData[i].command === "removePing") {
            let pingToRemove = allPing[jsonData[i].id]
            delete allPing[jsonData[i].id]
            removeFromMap(pingToRemove)
            console.log("removePing - ", jsonData[i])
        }
    }
})

export function sendMessage(message) {
    console.log(message)
    socket.send(message)
}

/*export function sendMessageVal(message) {
    console.log(message)
    socketVal.send(message)
}*/

function showEntityInfo(entity) {
    setSidebarEntity(entity.id);
    document.getElementById("entityDetails").style.width = "15%";
    document.getElementById("entityDetails").style.zIndex = "10";
    document.getElementById("entityTitle").innerHTML = entity.name + " - " + entity.id;
    document.getElementById("health").innerHTML = "Health: " + entity.health;
    const payloadsContainer = document.getElementById("payloadsContainer");
    while (payloadsContainer.firstChild) {
        payloadsContainer.removeChild(payloadsContainer.firstChild);
    }

    document.getElementById("manualHideShow").innerHTML = "Hide"

    disableButton(fireButton);
    disableButton(landButton);
    disableButton(takeoffButton);

    if (entity instanceof Drone) {
        enableButton(landButton);
        enableButton(takeoffButton);

        if (entity.role !== "ISR" && entity.role !== "Transport" && entity.role !== "Airliner") {
            enableButton(fireButton);
        }
    }

    if (entity.payloads) {
        for (let payload_index in entity.payloads) {
            const payload = entity.payloads[payload_index]
            console.log("Sidebar: ", payload_index, " - ", payload)
            var payloadDiv = document.createElement('div');
            payloadDiv.id = payload_index;
            payloadDiv.style.marginBottom = "1vh"
            var payloadTitle = document.createElement("h3");
            payloadTitle.style.margin = "0"
            payloadTitle.innerHTML = entity.payloads[payload_index].name + ": " + (entity.payloads[payload_index].current_quantity || "0") + "/" + entity.payloads[payload_index].max_quantity;
            document.getElementById("payloadsContainer").appendChild(payloadDiv);
            payloadDiv.appendChild(payloadTitle);
            if (entity.payloads[payload_index].name.toString().includes("Hangar")) {
                var hangarDiv = document.createElement('div');
                hangarDiv.style.marginLeft = "1vh"
                const hangar = entity.payloads[payload_index].carried_entities
                for (let drone in hangar) {
                    var droneDivWrapper = document.createElement("div");
                    var droneDiv = document.createElement("h3");
                    var releaseButton = document.createElement("button")

                    droneDivWrapper.style.margin = "0"

                    droneDiv.style.margin = "0"
                    droneDiv.innerHTML = "ID: " + hangar[drone]

                    releaseButton.innerHTML = "Release"
                    releaseButton.addEventListener("click", function() {
                        const message = JSON.stringify({
                            cmd: 'releaseDrone',
                            id: hangar[drone],
                            baseId: entity.id,
                            hangarId: Number(payload_index)
                        });
                        sendMessage(message)
                    })

                    droneDivWrapper.appendChild(droneDiv)
                    droneDivWrapper.appendChild(releaseButton)
                    hangarDiv.appendChild(droneDivWrapper)
                }
                payloadDiv.appendChild(hangarDiv);
            }

        }
        payloadsContainer.appendChild(payloadDiv);
    }

    const navContainer = document.getElementById("navContainer");
    while (navContainer.firstChild) {
        navContainer.removeChild(navContainer.firstChild);
    }
    let posDiv = document.createElement("h3");
    posDiv.id = "posDiv"
    posDiv.innerHTML = "LatLng: " + entity.getLatLng().lat + "," + entity.getLatLng().lng
    navContainer.appendChild(posDiv);

    if (entity.alt !== undefined) {
        let altDiv = document.createElement("h3");
        altDiv.innerHTML = "Alt: " + (entity.alt || 0)
        altDiv.id = "alt"
        navContainer.appendChild(altDiv);
        let velDiv = document.createElement("h3")
        velDiv.innerHTML = "Vel: " + (entity.vel || 0)
        velDiv.id = "vel"
        navContainer.appendChild(velDiv)
    }

    if (entity instanceof KnownEntity) {
        let confirmation = document.createElement("h3")
        if (entity.confirmed) {
            confirmation.innerHTML = "Presence of Entity is confirmed - it is likely <h2>not destroyed</h2>"
            confirmation.style.backgroundColor = "#005710"
        } else {
            confirmation.innerHTML = "Presence of Entity is unconfirmed - it is likely <h2>destroyed</h2>"
            confirmation.style.backgroundColor = "#730101"
        }
        confirmation.id = "confirmation"

        let confirmButton = document.createElement("button")
        confirmButton.innerHTML = "Confirm Presence"
        confirmButton.id = "confirmPresence"
        confirmButton.addEventListener("click", function() {
            console.warn(allPing)
            console.warn(allPing[entity.id])
            console.warn(allPing[entity.id] === undefined)
            console.warn(`Confirm Presence - ${allPing[entity.id]}`)
            if (allPing[entity.id]) {
                entity.confirmed = true
                confirmation.innerHTML = "Presence of Entity is confirmed - it is likely <h2>not destroyed</h2>"
                confirmation.style.backgroundColor = "#005710"
            } else {
                entity.confirmed = false
                confirmation.innerHTML = "Presence of Entity is unconfirmed - it is likely <h2>destroyed</h2>"
                confirmation.style.backgroundColor = "#730101"
            }
        })
        document.getElementById("payloadsContainer").appendChild(confirmation)
        document.getElementById("payloadsContainer").appendChild(confirmButton)
    }
}

function enableButton(button) {
    button.disabled = false;
    button.classList.remove('disabled');
    button.classList.add('enabled');
}

function disableButton(button) {
    button.disabled = true;
    button.classList.remove('enabled');
    button.classList.add('disabled');
}

const fireButton = document.getElementById("manualFire")
const landButton = document.getElementById("manualLand")
const takeoffButton = document.getElementById("manualTakeoff")
const hideShowButton = document.getElementById("manualHideShow")

fireButton.addEventListener("click", function() {
    let target = Number(prompt("Target", "0"))
    let air = Number(prompt("Air (1) | Gnd (0)", "0"))
    if (target > 0) {
        const message = JSON.stringify({
            cmd: 'fire',
            myId: getSidebarEntity(),
            targetId: target,
            air: air
        });
        sendMessage(message)
    }
})

landButton.addEventListener("click", function() {
    const message = JSON.stringify({
        cmd: 'land',
        myId: getSidebarEntity()
    });
    sendMessage(message)
})

takeoffButton.addEventListener("click", function() {
    const message = JSON.stringify({
        cmd: 'takeoff',
        myId: getSidebarEntity()
    });
    sendMessage(message)
})

hideShowButton.addEventListener("click", function() {
    const entity = allEntity[getSidebarEntity()]
    if (hideShowButton.innerHTML === "Hide") {
        entity.setOpacity(0)
        entity.getElement().style.pointerEvents = 'none';
        hideShowButton.innerHTML = "Show"
    } else {
        entity.setOpacity(1)
        entity.getElement().style.pointerEvents = 'auto';
        hideShowButton.innerHTML = "Hide"
    }
})