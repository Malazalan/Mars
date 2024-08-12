import {setSidebarEntity} from "./classes/entity.js";

let { sidebarEntity } = import("./classes/entity.js");

export var map = L.map('map', {
    zoomSnap: 0.25
}).setView([48.3667, -63.9606], 6.5);

map.addEventListener("click", function() {
    setSidebarEntity(null);
    document.getElementById("entityDetails").style.width = "0";
    document.getElementById("entityDetails").style.zIndex = "-1";
});

L.tileLayer('http://maps.bedge/tile/{z}/{x}/{y}.png', {maxZoom: 18}).addTo(map);

fetch("../static/labels.geo.json")
    .then((res) => res.text())
    .then((text) => {
        const parsed = $.parseJSON(text);
        L.geoJSON(parsed, {
            pointToLayer: function (feature, latlng) {
                switch (feature.properties.type) {
                    case "nation":
                        return L.marker(latlng, {
                            icon: L.divIcon({
                                className: 'leaflet-nation-labels',
                                html: feature.properties.name,
                            })
                        });
                    case "geographical":
                        return L.marker(latlng, {
                            icon: L.divIcon({
                                className: 'leaflet-geographical-labels',
                                html: feature.properties.name,

                            })
                        });
                    case "default":
                        return L.marker(latlng, {
                            icon: L.divIcon({
                                className: 'leaflet-map-labels',
                                html: feature.properties.name,
                            })
                        });
                }
            }
        }).addTo(map)
        ;
    })
    .catch((e) => console.error(e));

fetch("../static/borders.geo.json")
    .then((res) => res.text())
    .then((text) => {
        const parsed = $.parseJSON(text);
        L.geoJson(parsed, {
            style: function (feature) {
                return {color: feature.properties.color, fillOpacity: 0};
            }
        }).addTo(map);
    })
    .catch((e) => console.error(e));

const border = [{
    "type": "LineString",
    //"coordinates": [[47.909,-61.567],[47.909,-59.810],[49.009,-59.810],[49.009,-61.567]]
    "coordinates": [[-69.08571,45.202037],[-69.08571,50.910707],[-59.5136,50.910707],[-59.5136,45.202037],[-69.08571,45.202037]]
}];

const borderStyle = {
    "color": "#000000",
    "weight": 10,
    "opacity": 1
}

const hamptoniaStyle = {
    "color": "#d17878",
    "weight": 5,
    "opacity": 1
}

// Add the border to the map
L.geoJson(border, {
    style: borderStyle
}).addTo(map);

// Add a circle for Hamptonia with a 5km radius
L.circle([47.7515, -60.3971], {
    color: hamptoniaStyle.color,
    weight: hamptoniaStyle.weight,
    opacity: hamptoniaStyle.opacity,
    radius: 7000  // 5 km radius
}).addTo(map);


export function addToMap(entity) {
    entity.addTo(map)
    console.warn(`Add - ${entity}`)
}

export function removeFromMap(entity) {
    map.removeLayer(entity)
}