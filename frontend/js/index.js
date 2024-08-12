import { map } from "./map.js";
import {sendMessage, socket} from "./websocket.js";

import {Drone, sidebarEntity, allEntity} from "./classes/entity.js";

var selected = null;
var makingRoute = false
var numRoutes = 0
let allRoutes = [];
const allEntities = new Map()
/*
* {
*   {
*       id: drone,
*       id: drone
*   },
*   {
*       id: airbase,
*       id: airbase
*   }
* }
* */


window.addEventListener(('load'), function() {
    console.warn("Waiting for load")
    if (map) {
        console.log("onMapRClick() setup")
        map.on('contextmenu', onMapRClick);
    } else {
        console.error("Map isn't defined")
    }
    map.on('click', function() {
        selected = null;
    })

    document.getElementById('uploadFile').addEventListener('change', function (e) {
        let file = e.target.files[0];
        let reader = new FileReader();
        e.preventDefault();

        reader.onload = function (event) {
            try {
                let geojson = JSON.parse(event.target.result);
                let geojsonLayer = L.geoJson(geojson, {
                    style: function (feature) {
                        return { color: feature.properties.color || 'blue', fillOpacity: 0, weight: 3};
                    }
                }).addTo(map);

                let name = geojson.features[0].properties.name.split('Speed')[0];
                let distance = geojson.features[0].properties.distance;

                var entry = document.createElement("div");
                entry.className = "entry";

                var titleDeleteContainer = document.createElement("div");
                titleDeleteContainer.className = "entry-title-delete";

                var entryTitle = document.createElement("p");
                entryTitle.innerHTML = name;
                entryTitle.style.margin = "0";
                titleDeleteContainer.appendChild(entryTitle);

                var deleteEntry = document.createElement("button");
                deleteEntry.innerHTML = "Delete Entry";
                deleteEntry.addEventListener('click', function (e) {
                    map.removeLayer(geojsonLayer); // Remove the geojsonLayer from the map
                    entry.remove(); // Remove the entry div
                });
                titleDeleteContainer.appendChild(deleteEntry);

                entry.appendChild(titleDeleteContainer);

                var colorThicknessContainer = document.createElement("div");
                colorThicknessContainer.className = "entry-color-thickness";

                var colourPicker = document.createElement("input");
                colourPicker.type = "color";
                colourPicker.value = geojson.features[0].properties.color || '#0000ff'; // Default to blue if no color specified
                colourPicker.addEventListener('input', function (e) {
                    let selectedColor = e.target.value;
                    // Update the color of the GeoJSON layer
                    geojsonLayer.setStyle({
                        color: selectedColor
                    });
                });
                colorThicknessContainer.appendChild(colourPicker);

                var thicknessSlider = document.createElement("input");
                thicknessSlider.type = "range";
                thicknessSlider.min = 1;
                thicknessSlider.max = 10;
                thicknessSlider.step = 0.5;
                thicknessSlider.value = 3;
                thicknessSlider.addEventListener("input", function (e) {
                    geojsonLayer.setStyle({
                        weight: thicknessSlider.value
                    });
                });
                colorThicknessContainer.appendChild(thicknessSlider);

                entry.appendChild(colorThicknessContainer);

                var speedTimeContainer = document.createElement("div");
                speedTimeContainer.className = "entry-speed-time";

                var speedInput = document.createElement("input");
                speedInput.type = "number";
                speedInput.min = 1;
                speedInput.addEventListener('input', function (e) {
                    timeOutput.innerHTML = writeTimeNeatly(distance / speedInput.value);
                });
                speedTimeContainer.appendChild(speedInput);

                var timeOutput = document.createElement("strong");
                timeOutput.innerHTML = "Enter a speed";
                speedTimeContainer.appendChild(timeOutput);

                entry.appendChild(speedTimeContainer);

                document.getElementById("routes-container").appendChild(entry);
            } catch (error) {
                console.error("Invalid GeoJSON object", error);
            }
        };

        reader.onerror = function (error) {
            console.error("Error reading file", error);
        };

        if (file) {
            reader.readAsText(file);
        } else {
            console.error("No file selected");
        }
    });
})

/*map.on('contextmenu', function (e) {
    if (makingRoute) {
        for (let key in allRoutes) {
            if (allRoutes[key].active) {
                allRoutes[key].removePoint()
                allRoutes[key].newDistance()
            }
        }
    }
});

map.on('click', function(e) {
    if (!makingRoute) {
        var popup = L.popup()
        popup
            .setLatLng(e.latlng)
            .setContent(e.latlng.toString())
            .openOn(map);
    } else {
        for (let key in allRoutes) {
            if (allRoutes[key].active) {
                allRoutes[key].pushPoint(e.latlng, map)
                allRoutes[key].newDistance()
            }
        }
    }
})*/


function onMapRClick(e) {
    const entity = allEntity[sidebarEntity]
    if (!entity) {
        console.error("No aircraft selected");
    } else if (entity instanceof Drone) {
        console.log(`Moving ${entity.id} to ${e}`);
        let altitude = 0
        if (entity.role !== "BCT") {
            altitude = Number(prompt("Altitude", `${Math.ceil(entity.alt / 100) * 100}`))
        }
        const message = JSON.stringify({
            cmd: 'move',
            id: entity.id,
            lat: e.latlng.lat,
            lon: e.latlng.lng,
            alt: altitude
        });
        sendMessage(message)
    } else {
        console.warn(`Entity is a ${typeof entity}`)
    }
}

function writeTimeNeatly(time) {
    var hours, mins

    hours = Math.floor(time / 3600);
    time -= hours * 3600;

    mins = Math.floor(time / 60);
    time -= mins * 60;

    return (hours + "hr : " + mins + "mins : " + Math.floor(time) + "secs")
}

function selectDrone(tgt) { // set the selected aircraft to the clicked aircraft
    selected = tgt;
    console.log("Selected " + selected.name);
}

function deselectDrone() {
    selected = null;
}

let newRouteButton = document.getElementById("newRoute")
newRouteButton.addEventListener("click", function (e) {
    numRoutes++
    let geojsonLayer = L.geoJson().addTo(map);
    let startPoint = L.marker([0,0]).addTo(map)
        .bindPopup('Start')
        .closePopup
    let endPoint = L.marker([0,0]).addTo(map)
        .bindPopup('End')
        .closePopup

    var entry = document.createElement("div");
    let newRoute = new NewRoute(true, entry, geojsonLayer, map);
    allRoutes.push(newRoute);
    for (let allRoutesKey in allRoutes) {
        allRoutes[allRoutesKey].deactivate();
    }
    newRoute.activate();
    makingRoute = true;

    entry.className = "entry";
    entry.addEventListener("click", function (e) {
        if (!newRoute.active) {
            for (let allRoutesKey in allRoutes) {
                allRoutes[allRoutesKey].deactivate();
            }
            newRoute.activate();
            makingRoute = true;
        } else {
            newRoute.deactivate();
            makingRoute = false;
        }
    })

    var titleDeleteContainer = document.createElement("div");
    titleDeleteContainer.className = "entry-title-delete";

    var entryTitle = document.createElement("input");
    this.id = "entryTitle"
    entryTitle.value = "Route " + numRoutes;
    entryTitle.style.margin = "0";
    titleDeleteContainer.appendChild(entryTitle);

    var exportButton = document.createElement("button");
    exportButton.innerHTML = "Export Route";
    exportButton.addEventListener("click", function (e) {
        newRoute.exportRoute(entryTitle.value, colourPicker.value);
    })
    entry.appendChild(exportButton);

    var deleteEntry = document.createElement("button");
    deleteEntry.innerHTML = "Delete Entry";
    deleteEntry.addEventListener('click', function (e) {
        map.removeLayer(geojsonLayer); // Remove the geojsonLayer from the map
        map.removeLayer(newRoute.startPoint)
        map.removeLayer(newRoute.endPoint)
        entry.remove(); // Remove the entry div
        numRoutes--
        makingRoute = false;
    });
    titleDeleteContainer.appendChild(deleteEntry);

    entry.appendChild(titleDeleteContainer);

    var colorThicknessContainer = document.createElement("div");
    colorThicknessContainer.className = "entry-color-thickness";

    var colourPicker = document.createElement("input");
    colourPicker.type = "color";
    colourPicker.value = '#ffffff'; // Default to blue if no color specified
    colourPicker.addEventListener('input', function (e) {
        let selectedColor = e.target.value;
        // Update the color of the GeoJSON layer
        geojsonLayer.setStyle({
            color: selectedColor
        });
        newRoute.colour = selectedColor
    });
    colorThicknessContainer.appendChild(colourPicker);

    var thicknessSlider = document.createElement("input");
    thicknessSlider.type = "range";
    thicknessSlider.min = 1;
    thicknessSlider.max = 10;
    thicknessSlider.step = 0.5;
    thicknessSlider.value = 3;
    thicknessSlider.addEventListener("input", function (e) {
        geojsonLayer.setStyle({
            weight: thicknessSlider.value
        });
        newRoute.weight = thicknessSlider.value
    });
    colorThicknessContainer.appendChild(thicknessSlider);

    entry.appendChild(colorThicknessContainer);

    var speedTimeContainer = document.createElement("div");
    speedTimeContainer.className = "entry-speed-time";

    var speedInput = document.createElement("input");
    speedInput.type = "number";
    speedInput.min = 1;
    speedInput.addEventListener('input', function (e) {
        console.log(newRoute.distance)
        timeOutput.innerHTML = writeTimeNeatly(newRoute.distance / speedInput.value);
    });
    speedTimeContainer.appendChild(speedInput);

    var timeOutput = document.createElement("strong");
    timeOutput.innerHTML = "Enter a speed";
    speedTimeContainer.appendChild(timeOutput);

    entry.appendChild(speedTimeContainer);

    document.getElementById("routes-container").appendChild(entry);
})
