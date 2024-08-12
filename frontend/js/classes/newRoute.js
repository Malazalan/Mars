class NewRoute {
    constructor(active, entryDiv, layer, map) {
        this.active = active;
        this.entryDiv = entryDiv;
        this.layer = layer;
        this.route = [];
        this.distance = 0;
        this.startPoint = L.marker([0,0]).addTo(map).bindPopup("Start");
        this.startPoint._icon.style.filter = "hue-rotate(600deg)"
        this.endPoint = L.marker([0,0]).addTo(map).bindPopup("End");
        this.endPoint._icon.style.filter = "hue-rotate(150deg)"
        this.colour = '#ffffff';
        this.weight = 3
    }

    deactivate() {
        this.active = false
        this.entryDiv.style.backgroundColor = "#989898"
    }

    activate() {
        this.active = true
        this.entryDiv.style.backgroundColor = "#dcdbdb"
    }

    newDistance() {
        this.distance = 0;
        for (let i = 0; i < this.route.length - 1; i++) {
            this.distance += L.latLng(this.route[i]).distanceTo(L.latLng(this.route[i+1]));
        }
    }

    pushPoint(point) {
        this.route.push([point.lng, point.lat]);
        //this.route.push(point)
        console.log(this.route); // Check the route array
        if (this.route.length === 1) {
            this.startPoint.setLatLng([this.route[0][1],this.route[0][0]]).openPopup();
            this.startPoint.setOpacity(1)
        }
        if (this.route.length > 1) {
            this.endPoint.setLatLng([this.route[this.route.length-1][1],this.route[this.route.length-1][0]]).openPopup();
            this.endPoint.setOpacity(1);

            console.log(this.colour)

            var geojsonFeature = {
                "type": "Feature",
                "properties": {
                    "name": "blank",
                    "color": this.colour,
                    "distance": this.distance,
                    "weight": this.weight
                },
                "geometry": {
                    "type": "LineString",
                    "coordinates": this.route
                }
            };

            this.layer.clearLayers(); // Clear existing layers to redraw the updated route
            this.layer.addData(geojsonFeature);
            this.layer.setStyle({
                weight: this.weight,
                color: this.colour
            })
        }
    }

    removePoint() {
        if (this.route.length > 0) {
            if (this.route.length === 1) {
                this.startPoint.setOpacity(0)
            }
            this.route.splice(this.route.length - 1, 1)
            var geojsonFeature = {
                "type": "Feature",
                "properties": {
                    "name": "blank",
                    "color": "#000000",
                    "distance": this.distance
                },
                "geometry": {
                    "type": "LineString",
                    "coordinates": this.route
                }
            };
            this.layer.clearLayers(); // Clear existing layers to redraw the updated route
            this.layer.addData(geojsonFeature);
        }
        console.log(this.route.length)
        if (this.route.length > 1) {
            this.endPoint.setLatLng([this.route[this.route.length-1][1],this.route[this.route.length-1][0]]).openPopup();
            this.endPoint.setOpacity(1);
        } else {
            this.endPoint.setOpacity(0);
            this.endPoint.closePopup();
        }
        console.log(this.route)
    }

    exportRoute(name, colour) {
        var geoJSON = this.layer.toGeoJSON();
        geoJSON.features.forEach(feature => {
            feature.properties.name = name;
            feature.properties.color = colour;
        });
        var file = name + '.geo.json';
        saveAs(new File([JSON.stringify(geoJSON)], file, {
            type: "text/plain;charset=utf-8"
        }), file);
    }

}