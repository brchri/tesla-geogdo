# Change Log
All notable changes to this project will be documented in this file.

## [2.1.0] - 2024.01.01

### Added
- simple http api endpoints `/pause` and `/resume` on port `8555` to pause and resume garage door operations; see [README.md](/README.md) for details

### Changed

### Fixed
- potential geofence flapping issues, causing doors to operate incorrectly when just entering a close geofence or just leaving an open geofence

## [2.0.0] - 2023.12.21

**NOTE: This is a breaking change for config files. Please be sure to reference the [examples](/examples) directory for the most up to date config file structure when updating to this version!!**

### Added
- support for any location tracker that can publish to an mqtt broker; this enables support for any vehicle if the user can publish location data via an app like TeslaMate or OwnTracks on a smartphone

### Changed
- config file changes to support the feature that adds support for any location tracker that can publish to an mqtt broker; see the [examples](/examples) folder for the most up to date examples of config file structure

### Fixed
- minor debug message formatting

## [1.0.0] - 2023.12.05

### Added
- support for homebridge controlled gdo's
- fully releasing to 1.0.0 for proper semantic versioning of releases going forward

### Changed

### Fixed

## [0.0.2] - 2023.12.02

### Added
- support for home assistant controlled gdo's (#7)
 
### Changed
 
### Fixed
- mitigate geofence flapping when status checks and operation cooldowns are disabled (#6)
 
## [0.0.1] - 2023.11.10
 
### Added
- initial release since fork of Tesla-YouQ
- Support for ratgdo, http, and mqtt gdo's
 
### Changed
- Removed support for MyQ gdo's due to MyQ disabling 3rd party API access
 
### Fixed
