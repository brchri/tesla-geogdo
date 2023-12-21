# Tesla-GeoGDO
**NOTE: As of v2.0.0, this app works with any location tracking that can publish to an MQTT broker (e.g. TeslaMate for Teslas, or the OwnTracks app for smartphones); it is no longer limited to tracking Teslas. See [Prerequisites](#prerequisite) for details.**

A lightweight app that will operate your smart Garage Door Openers (GDOs) based on the position of your location tracker (e.g. Tesla vehicle or phone location), automatically closing when you leave, and opening when you return. Supports multiple vehicles, location trackers (see [Prerequisites](#prerequisite)), geofence types, and smart GDO devices.

<!-- TOC -->

- [Tesla-GeoGDO](#tesla-geogdo)
  - [Supported Smart Garage Door Openers](#supported-smart-garage-door-openers)
  - [Prerequisite](#prerequisite)
  - [How to use](#how-to-use)
    - [Docker](#docker)
    - [Supported Environment Variables](#supported-environment-variables)
  - [Notes](#notes)
    - [Geofence Types](#geofence-types)
      - [Circular Geofence](#circular-geofence)
      - [TeslaMate Defined Geofence](#teslamate-defined-geofence)
      - [Polygon Geofence](#polygon-geofence)
    - [Operation Cooldown](#operation-cooldown)
  - [Credits](#credits)

<!-- /TOC -->

## Supported Smart Garage Door Openers
### Current
* Any Garage Door Opener managed by [Home Assistant](https://www.home-assistant.io/)
  * Controlled by proxying commands through Home Assistant
* Any Garage Door Opener managed by [Homebridge](https://homebridge.io/)
  * Controlled by proxying commands through Homebridge
* Native [ratgdo](https://paulwieland.github.io/ratgdo/) (MQTT-based firmware)
  * Support also available for ratgdo using ESP Home firmware and managed by Home Assistant or Homebridge
* Generic MQTT Controlled Smart Garage Door Openers
* Generic HTTP Controlled Smart Garage Door Openers
### Deprecated:
* MyQ
  * No longer supported due to MyQ API changes blocking 3rd party integrations
### Potentially Upcoming
* Native [Meross](https://www.meross.com/en-gc/product) (if I get my hands on one or can use someone else's package to incorporate)
  * Meross GDO's that are managed by Home Assistant or Homebridge *are currently supported*

## Prerequisite
As of v2.0.0, this app supports any location tracker that can publish to an MQTT broker. For tesla vehicles, the easiest way to accomplish this is to use the [MQTT capabilities](https://docs.teslamate.org/docs/integrations/mqtt) of [TeslaMate](https://github.com/adriankumpf/teslamate). For other users, you can use an app like [OwnTracks](https://owntracks.org/) on your phone to publish your location to an MQTT broker for tracking.

## How to Use
### Docker
This app is provided as a docker image. You will need to create a `config.yml` file (please refer to the [examples directory](/examples) or the simplified [config.simple.example.yml](config.simple.example.yml)), edit it appropriately (***make sure to preserve the leading spaces, they are important***), and then mount it to the container at runtime. For example:

```bash
# see docker compose example below for parameter explanations
docker run \
  -e TZ=America/New_York \
  -v /etc/tesla-geogdo:/app/config \
  brchri/tesla-geogdo:latest
```

Or you can use a docker compose file like this:

```yaml
version: '3.8'
services:
  tesla-geogdo:
    image: brchri/tesla-geogdo:latest
    container_name: tesla-geogdo
    environment:
      - TZ=America/New_York # optional, sets timezone for container
    volumes:
      - /etc/tesla-geogdo:/app/config # required, mounts folder containing config file(s) into container
    restart: unless-stopped
```

### Supported Environment Variables
The following Docker environment variables are supported but not required.
| Variable Name | Type | Description |
| ------------- | ---- | ----------- |
| `CONFIG_FILE` | String (Filepath) | Path to config file within container |
| `TRACKER_MQTT_USER` | String | User to authenticate to MQTT broker. Can be used instead of setting `global.tracker_mqtt_settings.connection.user` in the `config.yml` file |
| `TRACKER_MQTT_PASS` | String | Password to authenticate to MQTT broker. Can be used instead of setting `global.tracker_mqtt_settings.connection.pass` in the `config.yml` file |
| `DEBUG` | Bool | Increases output verbosity |
| `TESTING` | Bool | Will perform all functions *except* actually operating garage door, and will just output operation *would've* happened |
| `TZ` | String | Sets timezone for container |

## Notes
### Geofence Types
You can define 3 different types of geofences to trigger garage operations. You must configure *one and only one* geofence type for each garage door. Each geofence type has separate `open` and `close` configurations (though they can be set to the same values). This is useful for situations where you might want a smaller geofence that closes the door so you can visually confirm it's closing, but you want a larger geofence that opens the door so it will start sooner and be fully opened when you actually arrive.

Note you do not need to define both `open` and `close` for a geofence, you may only define one or the other if you don't wish to have Tesla-GeoGDO both open and close your garage.

#### Circular Geofence
This is the simplist geofence to configure. You provide a latitude and longitude coordinate as the center point, and the distance from the center point to trigger the garage action (in kilometers). You can use a tool such as [FreeMapTools](https://www.freemaptools.com/radius-around-point.htm) to determine what the center latitude and longitude coordinates are, as well as how big your want your radius to be. An example of a garage door configured with this type of geofence would look like this:

```yaml
garage_doors:
  - geofence:
      type: circular
      settings:
        center:
          lat: 46.19290425661381
          lng: -123.79965087116439
        close_distance: .013
        open_distance: .04
    opener:
      type: ratgdo
      mqtt_settings:
        connection:
          host: localhost
          port: 1883
        prefix: home/garage/Main
    trackers:
      - id: 1
        lat_topic: teslamate/cars/1/latitude
        lng_topic: teslamate/cars/1/longitude
```
This would produce two circular geofences (open and close) that look like this:

![image](https://github.com/brchri/tesla-geogdo/assets/126272303/60d9c60b-095d-478b-9418-6a670a225daf)

Under this configuration, your garage would start to open when you *entered* the `open_distance` area, and would start to close as you *exit* the `close_distance` area.

#### TeslaMate Defined Geofence
You can choose to use geofences defined in TeslaMate. To define these geofences, go to your TeslaMate page and click `Geo-Fences` at the top, and create a new fence (or reference your existing fences). Some notes about using TeslaMate Defined Geofences:
* TeslaMate does not update its geofence calculations in realtime. *This will cause delays in your garage door operations*.
* You cannot define overlapping geofences in TeslaMate, as it will cause TeslaMate to behave unexpectedly as it cannot determine which Geofence you should be in when you're in more than one. This means you cannot define separate open and close geofences and should only use a single geofence.
* You must configure TeslaMate to have a "default" geofence when no defined geofences apply. You do this by configuring an environment variable for TeslaMate, such as `DEFAULT_GEOFENCE=not_home`.
* In general, it is not recommended to use this method as it is the least reliable due to how TeslaMate updates the Geofence data.

An example of a garage door configured with this type of geofence would look like this:

```yaml
garage_doors:
  - geofence:
      type: teslamate
      settings:
        close_trigger:
          from: home
          to: not_home
        open_trigger:
          from: not_home
          to: home
    opener:
      type: ratgdo
      mqtt_settings:
        connection:
          host: localhost
          port: 1883
        prefix: home/garage/Main
    trackers:
      - id: 1
        complex_topic:
          topic: owntracks/owner
          lat_json_key: lat
          lng_json_key: lon
```

#### Polygon Geofence
This is the most customizable method of defining a geofence, which allows you to specifically define a polygonal geofence using a list of latitude and longitude coordinates. You can use a tool like [geojson.io](https://geojson.io/) to assist with creating a geofence and providing latitude and longitude points. **NOTE:** Using tools like this often specify longitude *before* latitude in the output, as defined by the [KML spec](https://developers.google.com/kml/documentation/kmlreference?csw=1#coordinates). Be sure you're identifying the latitude and longitude correctly.

An example of a garage door configured with this type of geofence would look like this:

```yaml
garage_doors:
  - geofence:
      type: polygon
      settings:
        open:
          - lat: 46.193245921812746
            lng: -123.7997972320742
          - lat: 46.193052416203386
            lng: -123.79991877106825
          - lat: 46.192459275200264
            lng: -123.8000342331126
          - lat: 46.19246067743231
            lng: -123.8013205208015
          - lat: 46.19241300151987
            lng: -123.80133064905115
          - lat: 46.192411599286004
            lng: -123.79997751491551
          - lat: 46.1927747765306
            lng: -123.79954200018626
          - lat: 46.19297669643191
            lng: -123.79953592323656
          - lat: 46.193245921812746
            lng: -123.7997972320742
        close:
          - lat: 46.192958467582514
            lng: -123.7998033090239
          - lat: 46.19279440766502
            lng: -123.7998033090239
          - lat: 46.19279440766502
            lng: -123.79950958978756
          - lat: 46.192958467582514
            lng: -123.79950958978756
          - lat: 46.192958467582514
            lng: -123.7998033090239
    opener:
      type: ratgdo
      mqtt_settings:
        connection:
          host: localhost
          port: 1883
        prefix: home/garage/Main
    trackers:
      - id: 1
        lat_topic: teslamate/cars/1/latitude
        lng_topic: teslamate/cars/1/longitude
```

Or, using a tool referenced above or any other of your choosing, you can generate and download a KML file containing your polygon geofences instead of manually defining the points in your config file. Be sure that the KML file is in a mounted volume and accessible within the container. Within your KML file, you *must* define a `name` element within each `Placemark` element for each geofence, with the value `open` or `close` accordingly. Please see the [polygon_map.kml](resources/polygon_map.kml) file for an example.

An example of a garage door configured this way would look like this:

```yaml
garage_doors:
  - geofence:
      type: polygon
      settings:
        kml_file: config/polygon_geofences.kml
    opener:
      type: ratgdo
      mqtt_settings:
        connection:
          host: localhost
          port: 1883
        prefix: home/garage/Main
    trackers:
      - id: 1
        lat_topic: teslamate/cars/1/latitude
        lng_topic: teslamate/cars/1/longitude
```

Either of these configs would produce two polygonal geofences (open and close) that look like this:

![image](https://github.com/brchri/tesla-geogdo/assets/126272303/2dbcb375-0425-44af-b8c0-3f7e79762b54)

Under this configuration, your garage would start to open when you *entered* the `open` area, and would start to close as you *exit* the `close` area.

### Operation Cooldown
There's a configurable `cooldown` parameter in the `config.yml` file's `global` section that will allow you to specify how many minutes Tesla-GeoGDO should wait after operating a garage door before it attemps any further operations. This helps prevent potential flapping if that's a concern.

## Credits
* [TeslaMate](https://github.com/adriankumpf/teslamate)
* [Ratgdo](https://paulwieland.github.io/ratgdo/)
* [@DjAnu](https://github.com/DjAnu) for the idea to expand beyond Tesla vehicles and help with testing
