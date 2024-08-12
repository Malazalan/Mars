// Global variables
import {Entity} from "./entity";

import {socket} from '../websocket.js'


export function importDrone(latlng, planeType, uid, IFF, callsign, nation, map) {
    const iconSrc = "../../static/icons/" + planeType.toString().toLowerCase() + ".svg";
    const newIcon = new L.divIcon({
        html: `<img src="${iconSrc}" alt="${planeType.toString().toLowerCase()}" style="width: 20px; height: 20px; background: transparent">`,
        iconSize: [20,20],
        popupAnchor: [0, -20]
    })

    const newDrone = new Drone(
        //[47.426890,-61.775058],
        latlng,
        newIcon,
        uid,
        IFF,
        callsign,
        planeType,
        nation,
        true
    )

    console.log(newDrone)
    console.log(map)

    allAircraft[uid] = newDrone
    newDrone.addTo(map);
    newDrone.bindPopup(newDrone.callsign);
    console.log(newDrone);

    //Functions
    newDrone.on("click", function () {
        selectAircraft(newDrone)
    })
    newDrone.on("contextmenu", function () {
        console.log("HELLO")
    })
}