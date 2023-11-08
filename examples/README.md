# Examples
This folder contains various `config.*.*.yml` files to show various configuration options. For your `config.yml` file, you will need:
* A `global` config section as shown in each file
* 1 or more `garage_door` configs defined, each with:
  * 1 and only 1 `geofence` config
  * 1 and only 1 `opener` config
  * 1 or more `cars` identified by their `teslamate_car_id`

In the `garage_doors` section, each garage door can have a different geofence type and configuration from the other defined doors, so you can mix and match the different config types from the config examples in this folder. This is also true for the openers; each garage door can have an opener with a unique type and settings compared to the openers for the other garage doors.

Please see the [config.multiple.yml](/examples/config.multiple.yml) file for a sample config file showing different geofence and opener configurations for different doors.

The example files will be named in the following format for ease of reference: `config.<geofence_type>.<opener_type>.yml`. Again, you can combine any `geofence_type` with any `opener_type`, the combinations in the examples are arbitrary.