# Mercury
*[**Mercury**](https://en.wikipedia.org/wiki/Mercury_(mythology)) - The Roman God of 
communication and the messenger of the Gods.*

Mercury is an open soruce drone situational awareness tool for use within the Blue Edge 
battlespace. Within this repository you will find all necessary code for building Mercury and 
Enceladus (an accompanying app for route planning).

---
## Contents
<!-- TOC -->
* [Mercury](#mercury)
  * [Contents](#contents)
  * [Features](#features)
  * [Enceladus](#enceladus)
  * [File Structure](#file-structure)
  * [File Options](#file-options)
    * [compose.yaml](#composeyaml)
    * [conf.json](#confjson)
    * [Dockerfile](#dockerfile)
    * [Enceladus/Dockerfile](#enceladusdockerfile)
    * [known_entities.json](#known_entitiesjson)
  * [How to Set Up](#how-to-set-up)
  * [Known Issues](#known-issues)
  * [Advised Next Steps](#advised-next-steps)
  * [Planned Work](#planned-work)
<!-- TOC -->

---
## Features
- Live mapping of controlled entities on the map
- Live updating of entity's payload (fuel, ammunition, carried entities, etc.)
- Live updating of entities being released from hangars
- Automatic map population of RADAR pings on the map
- Populating the map with static entities from [known_entities.json](#known_entitiesjson)
---
## Enceladus
*[**Enceladus**](https://en.wikipedia.org/wiki/Enceladus_(Giant)) - A giant from greek mythology 
that was created to kill Athena.*

Enceladus is a tool for plotting and viewing routes on the battlespace map. It is an extension 
of the [LeafletJS](https://leafletjs.com/) library - a tool for viewing interactive maps. To add 
a route to the map, it must be a *.geo.json* filetype. For clarification on how to format a *.
geo.json*, create a route with Enceladus and view the contents.
---
## File Structure
In order for Mercury to run correctly, make sure you structure your directory to match the below 
format. To avoid errors when importing files, see the [file options](#file-options) section.
```
.
├── compose.yaml
├── conf.json
├── Dockerfile
├── Enceladus
│   └── Dockerfile
└── known_entities.json
```
---
## File Options

### compose.yaml
```yaml
services:
  mercury:
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - "HOST IP:HOST PORT:MERCURY PORT (defined in Dockerfile)"
  enceladus:
    build:
      context: .
      dockerfile: Enceladus/Dockerfile
    ports:
      - "HOST IP:HOST PORT:3000" #HOST:DOCKER - don't change the docker port
```
The docker compose file stores all runtime variables for building & starting the Mercury and 
Enceladus Dockerfiles.

**Notes**:
- `HOST IP` for `mercury` and `enceladus` are **optional** and should be either the IP address of 
  your network device (e.g. *wlan0*, *enp0*, *wg0*), *localhost*, *127.0.0.1*, or left blank 
  (default is *0.0.0.0*)
- `HOST PORT` should be whichever port you want to use to access Mercury and Enceladus on with 
  your browser. **The ports must be different**
- `MERCURY PORT` must be the same as the `PORT` defined in [conf.json](#confjson)

### conf.json
```json
{
  "MERCURY_PORT": "PORT",
  "MERCURY_IP": "HOST IP",
  "WS_SEND_DELAY": SEND DELAY (in ms),
  "PING_EXPIRY": PING EXPIRY (in s),
  "0": {
    "ICARUS_SERVER_IP": "172.20.1.11" or "ICARUS SERVER IP",
    "ICARUS_SERVER_PORT": 4567 or ICARUS SERVER PORT,
    "NATION": "NATION"
  },
  ...,
  "N": {
    "ICARUS_SERVER_IP": "172.20.1.11" or "ICARUS SERVER IP",
    "ICARUS_SERVER_PORT": 4567 or ICARUS SERVER PORT,
    "NATION": "NATION"
  }
}
```
conf.json stores all configuration options for Mercury. All fields within conf.json **must** be 
filled or Mercury will throw errors.

**Notes**:
- `PORT` is the port that Mercury will use for hosting the webpage and websocket connection. It 
  must be the same as the `MERCURY PORT` defined in [compose.yaml](#composeyaml)
- `HOST IP` is the IP that Mercury is hosted on. It must be the same as `HOST IP` defined in 
  [compose.yaml](#composeyaml)
  - This option will not affect the entities you can see, only the colour of your entities
- `SEND DELAY` should be how frequently you want the backend websocket to send updates to your 
  frontend. If you experiencing delays with Websocket, increase this value and then reload Mercury
- `PING EXPIRY` is the time it takes for a RADAR ping to be removed from the map from it's most 
  recent ping. E.g. if `PING EXPIRY` is 60, then the ping will be removed from the map 60 
  seconds after the last time that entity was pinged
- `"0"` is the object storing the details of your first Icarus server. You **must** have at 
  least one
- `"N"` is the object storing other Icarus servers. Any additional servers **must** be 
  sequentially numbered (i.e. if you have 2 servers, you would have `"0"` and `"1"`)
  - `ICARUS_SERVER_IP` should either be the test range (above) or the IP address of your Icarus server
  - `ICARUS_SERVER_PORT` should either be the test range (above) or the port of your Icarus server
  - `NATION` should be your nation (Malazan, Valinor, Halcyon, Gallifrey, Civilian)

### Dockerfile
```dockerfile
FROM golang:1.22 AS builder
LABEL authors="alan"
WORKDIR /app

# Clone the latest Mercury version
RUN git config --global http.sslVerify false
RUN git clone https://gitlab.bedge/whoKnows/mercury.git .
RUN cat frontend/static/icons/isr.svg

# DEPRECATED - Change the current conf.json file to the default parameters
RUN sed -i "s|DOCKER_MERCURY_PORT|3001|g" /app/conf/conf.json
RUN sed -i "s|NATION_HERE|Malazan|g" /app/conf/conf.json

# Copy a conf.json file into the container
COPY conf.jso[n] known_entities.json Dockerfile conf/
RUN cat conf/conf.json
RUN rm conf/Dockerfile

# Download the go modules & build
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o mercury

# Verify all files are present
RUN ls -la /app

FROM ubuntu:latest AS runner

# Expose the ports
EXPOSE MERCURY PORT
EXPOSE ICARUS PORT
WORKDIR /app

# Copy over the necessary files from the builder
RUN mkdir -p /app/conf /app/frontend
COPY --from=builder /app/mercury /app/mercury
COPY --from=builder /app/conf/* /app/conf/
COPY --from=builder /app/frontend/ /app/frontend/

# Verify the files are present
RUN ls /app/conf
RUN ls /app/frontend
RUN cat /app/conf/conf.json

# Start Mercury
ENTRYPOINT ["/app/mercury"]
```
The Mercury Dockerfile will retrieve the most recent Dockerfile and build the go files.

**Notes**:
- `MERCURY PORT` should be the port that Mercury is hosting the web app and websocket on. This 
  must be the same as the ports defined in [compose.yaml](#composeyaml) and [conf.json](#confjson)
- `ICARUS PORT` should be the port that your Icarus server has exposed. This must be the same as 
  the port defined in [conf.json](#confjson)
### Enceladus/Dockerfile
```dockerfile
FROM golang:1.22 AS builder
LABEL authors="alan"
WORKDIR /app

# Clone the latest Mercury version
RUN git config --global http.sslVerify false
RUN git clone https://gitlab.bedge/whoKnows/enceladus.git .

# Download the go modules & build
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o enceladus

FROM ubuntu:latest AS runner

# Expose the port
WORKDIR /app

# Copy Enceledus from the builder
COPY --from=builder /app/enceladus /app/enceladus
COPY --from=builder /app/frontend/ /app/frontend/

# Start Enceladus
ENTRYPOINT ["/app/enceladus"]
```
The Enceladus Dockerfile will retrieve the most recent Dockerfile and build the go files.

### known_entities.json
```json
{
  "ID": {
    "type": "TYPE",
    "name": "NAME",
    "lat": XX.XX...,
    "lon": XX.XX...,
    "nation": "NATION"
  },
  ...
}
```
known_entities.json is used to both populate the map with any known static structure, and add 
additional details to any RADAR pings. If an entry in this file has a latitude and longitude, 
Mercury will add it to the map. If it does not, then Mercury will use it to add additional 
details to any RADAR pings (e.g. nation colour).

**Notes**
- `ID` - the ID of the entity you are referring to
- `type` - **optional** - If known, the type of the entity (e.g. Airbase, Depot, Fort, etc.)
- `lat` - ***semi*-optional** - If known, the latitude of an entity. This is only to be used for 
  any static entities (e.g. airbases, harbours) and not dynamic entities (e.g. drones)
- `lon` - ***semi*-optional** - If known, the longitude of an entity. This is only to be used for
  any static entities (e.g. airbases, harbours) and not dynamic entities (e.g. drones)
- `NATION` - **optional** - The nation of the entity. This will be used to colour the RADAR ping 
  / static entity on the map

---
## How to Set Up
1. Create a directory to store the files (`mkdir Mercury`)
2. Populate the files according to the instructions in [file options](#file-options)
3. Run `docker compose up --build`
   - This may cause errors depending on your setup. If you have issues, try:
   - `sudo docker compose up --build`
   - `docker-compose up --build`
   - `sudo docker-compose up --build`
4. Access Mercury / Enceladus on the ports you chose

---
## Known Issues
- Known_entities.json isn't showing on the map
  - If known_entities.json isn't working or you've updated the icons and they haven't changed, 
    it's likely because your browser has cached the old Mercury frontend.
  - Delete your cache and re-run Mercury to solve this
- Docker can't resolve gitlab.bedge
  - Run `sudo systemctl restart docker`

---
## Advised Next Steps
**SECURE MERCURY**!!!

Mercury is insecure by design. Consider how and where Mercury is exposed and take steps to add 
additional security.

---
## Planned Work
- Dead drones are removed from the map
- Pull data from multiple Icarus Servers
- Allow the addition of certs for mTLS connections to the Icarus Server
