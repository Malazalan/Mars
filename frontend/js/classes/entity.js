import { addToMap } from "../map.js";

export var allEntity = {}
export var sidebarEntity = null;
export var allPing = {}
export var allStaticEntity = {}
export var showDrones = true
export var showBases = true
export var showPings = true

export const entityTypes = {
    "Multi": "Drone",
    "ISR": "Drone",
    "Airbase": "Airbase",
    "Fighter": "Drone",
    "Bomber": "Bomber",
    "Depot": "Depot",
    "Transport": "Drone",
    "Airliner": "Drone"
}

export function setSidebarEntity(newValue) {
    sidebarEntity = newValue
}

export function getSidebarEntity() {
    return sidebarEntity
}

/*const nationColours = {
    "Malazan": "#0000ff",
    "Valinor": "#ff0000",
    "Halcyon": "#00ff00",
    "Gallifrey": "#ff00ff",
    "Civilian": "#ffffff"
}*/

const nationFilters = {
    "Malazan": "invert(16%) sepia(98%) saturate(6887%) hue-rotate(252deg) brightness(72%) contrast(139%)", // Blue
    "Malazan-West": "invert(16%) sepia(98%) saturate(6887%) hue-rotate(252deg) brightness(62%) contrast(119%)",
    "Malazan-East": "invert(16%) sepia(98%) saturate(6887%) hue-rotate(252deg) brightness(82%) contrast(159%)",
    "Valinor": "invert(19%) sepia(92%) saturate(5411%) hue-rotate(354deg) brightness(106%) contrast(104%)", // Red
    "Valinor-West": "invert(19%) sepia(92%) saturate(5411%) hue-rotate(354deg) brightness(90%) contrast(94%)",
    "Valinor-East": "invert(19%) sepia(92%) saturate(5411%) hue-rotate(354deg) brightness(116%) contrast(114%)",
    "Halcyon": "invert(58%) sepia(82%) saturate(2137%) hue-rotate(91deg) brightness(103%) contrast(101%)", // Green
    "Gallifrey": "invert(29%) sepia(100%) saturate(6264%) hue-rotate(285deg) brightness(84%) contrast(123%)", // Purple
    "Civilian": "invert(71%) sepia(30%) saturate(3592%) hue-rotate(357deg) brightness(98%) contrast(108%)", // Orange
    "Hamptonia": "brightness(0) saturate(100%) invert(85%) sepia(8%) saturate(5533%) hue-rotate(301deg) brightness(100%) contrast(73%);"
}


export function genIcon(entityType, nation, name) {
    if (entityTypes[entityType] === "Undefined") {
        entityType = "unknown"
    }
    const iconSrc = "../../static/icons/" + entityType.toString().toLowerCase() + ".svg";

    //const colouredSVG = iconSrc.replace(/fill="[^"]*"/g, `fill="${nationColours[nation]}"`);

    if (entityTypes[entityType] === "Drone") {
        return new L.divIcon({
            html: `<div class="icon-container" style="position: relative; transition: transform 0.2s;">
               <h6 style="margin: 0; height: 1.2em;"></h6>
               <img src="${iconSrc}" alt="icon" style="width: 23px; height: 23px; filter: ${nationFilters[nation]}; margin: 0">
               <h6 style="margin: 0; height: 1.2em;">${name}</h6>
               </div>
               <style>
                   .icon-container:hover {
                       transform: scale(1.5); /* Increase size on hover */
                       z-index: 1000; /* Bring to front on hover */
                   }
               </style>`,
            iconSize: [23, 46],
            popupAnchor: [0, -23]
        });
    } else {
        return new L.divIcon({
            html: `<div class="icon-container" style="position: relative; transition: transform 0.2s;">
               <h6 style="margin: 0; height: 1.2em;">${name}</h6>
               <img src="${iconSrc}" alt="icon" style="width: 23px; height: 23px; filter: ${nationFilters[nation]}; margin: 0">
               <h6 style="margin: 0; height: 1.2em;"></h6>
               </div>
               <style>
                   .icon-container:hover {
                       transform: scale(1.5); /* Increase size on hover */
                       z-index: 1000; /* Bring to front on hover */
                   }
               </style>`,
            iconSize: [23, 46],
            popupAnchor: [0, -23]
        });
    }
}

export class Entity extends L.Marker {
    constructor(lat, lng, icon, id, name, role, nation, alive, payloads, health) {
        super(L.latLng(lat, lng), {icon: icon});
        this.id = id;
        this.name = name;
        this.role = role;
        this.nation = nation;
        this.alive = alive;
        this.payloads = payloads
        this.health = health
    }

    importEntity() {
        addToMap(this)
        //Functions
        /*newDrone.on("click", function () {
            selectAircraft(newDrone)
        })
        newDrone.on("contextmenu", function () {
            console.log("HELLO")
        })*/
    }

    setRotation(angle) {
        const iconElement = this.getElement();
        if (iconElement) {
            iconElement.style.transform = `rotate(${angle}deg)`;
            iconElement.classList.add('rotated-marker');
        }
    }
}

export class KnownEntity extends Entity {
    constructor(lat, lng, icon, id, name, role, nation, alive, payloads) {
        super(lat, lng, icon, id, name, role, nation, alive, payloads);
        this.confirmed = false;
    }
}

export class AirBase extends Entity {
    constructor(lat, long, icon, id, name, role, nation, alive, payloads, health) {
        super(lat, long, icon, id, name, role, nation, alive, payloads, health);
    }

    releaseDrone(drone) {
        this.storing.delete(drone.id);
        drone.storedIn = null;
    }

    storeDrone(drone) {
        this.storing.set(drone.id, drone);
        drone.storedIn = this;
    }
}

export class Drone extends Entity {
    constructor(lat, lng, icon, id, name, role, nation, alive, alt, payloads, vel, health) {
        super(lat, lng, icon, id, name, role, nation, alive, payloads, health);
        this.alt = alt;
        this.vel = vel;
    }
}

export class Ping extends L.Marker {
    constructor(lat, lng, icon, id, role, nation) {
        super(L.latLng(lat, lng), {icon: icon});
        this.id = id;
        this.role = role;
        this.nation = nation;
    }
}

document.getElementById("hideDrones").addEventListener("click", function() {
    if (showDrones) {
        for (const [id, entity] of Object.entries(allEntity)) {
            if (entity instanceof Drone) {
                console.log(`Hiding ${id}`)
                entity.setOpacity(0)
                entity.getElement().style.pointerEvents = 'none';
            }
        }
        showDrones = false
        document.getElementById("hideDrones").innerHTML = "Show Drones"
    } else {
        for (const [id, entity] of Object.entries(allEntity)) {
            if (entity instanceof Drone) {
                console.log(`Showing ${id}`)
                entity.setOpacity(1)
                entity.getElement().style.pointerEvents = 'auto';
            }
        }
        showDrones = true
        document.getElementById("hideDrones").innerHTML = "Hide Drones"
    }
})

document.getElementById("hideStatic").addEventListener("click", function() {
    if (showBases) {
        for (const [id, entity] of Object.entries(allEntity)) {
            if (!(entity instanceof Drone)) {
                console.log(`Hiding ${id}`)
                entity.setOpacity(0)
                entity.getElement().style.pointerEvents = 'none';
            }
        }
        showBases = false
        document.getElementById("hideStatic").innerHTML = "Show Buildings"
    } else {
        for (const [id, entity] of Object.entries(allEntity)) {
            if (!(entity instanceof Drone)) {
                console.log(`Showing ${id}`)
                entity.setOpacity(1)
                entity.getElement().style.pointerEvents = 'auto';
            }
        }
        showBases = true
        document.getElementById("hideStatic").innerHTML = "Hide Buildings"
    }
})

document.getElementById("hidePings").addEventListener("click", function() {
    if (showPings) {
        for (const [id, ping] of Object.entries(allPing)) {
            console.log(`Hiding ${id} | ${ping instanceof Ping}`)
            console.log(typeof ping)
            ping.setOpacity(0)
            ping.getElement().style.pointerEvents = 'none';
        }
        showPings = false
        document.getElementById("hidePings").innerHTML = "Show Pings"
    } else {
        for (const [id, ping] of Object.entries(allPing)) {
            console.log(ping)
            console.log(`Showing ${id} = ${ping instanceof Ping}`)
            console.log(typeof ping)
            ping.setOpacity(1)
            ping.getElement().style.pointerEvents = 'auto';
        }
        showPings = true
        document.getElementById("hidePings").innerHTML = "Hide Pings"
    }
})
