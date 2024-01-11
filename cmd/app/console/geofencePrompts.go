package console

import (
	"fmt"
	"regexp"
	"strconv"
)

func runCircularGeofencePrompts() interface{} {
	fmt.Println("We will now work on configuring your circular geofence for this garage door.")

	geofence := map[string]interface{}{}
	latResponse := promptUser(question{
		prompt:                 "What is the latitude of the center of your circular geofence?",
		validResponseRegex:     "^(-)?\\d+\\.\\d+$",
		invalidResponseMessage: "Please enter a valid latitude, such as 123.456 or -123.456",
	})
	lngResponse := promptUser(question{
		prompt:                 "What is the longitude of the center of your circular geofence?",
		validResponseRegex:     "^(-)?\\d+\\.\\d+$",
		invalidResponseMessage: "Please enter a valid longitude, such as 123.456 or -123.456",
	})
	closeResponse := promptUser(question{
		prompt:                 "How far (in km) should you move away from the center point to trigger a garage close (leave blank to skip automatically closing garage)? []",
		validResponseRegex:     "^(\\d*\\.?\\d+)?$",
		invalidResponseMessage: "Please enter a valid distance in km",
	})
	openResponse := promptUser(question{
		prompt:                 "How close (in km) should you be to the center point to trigger a garage open (leave blank to skip automatically opening garage)? []",
		validResponseRegex:     "^(\\d*\\.?\\d+)?$",
		invalidResponseMessage: "Please enter a valid distance in km",
	})

	fmt.Println("Configuration of circular geofence for this garage door is complete, moving on...")

	lat, _ := strconv.ParseFloat(latResponse, 64)
	lng, _ := strconv.ParseFloat(lngResponse, 64)

	settings := map[string]interface{}{
		"center": map[string]interface{}{
			"lat": lat,
			"lng": lng,
		},
	}
	if len(closeResponse) > 0 {
		closeDistance, _ := strconv.ParseFloat(closeResponse, 64)
		settings["close_distance"] = closeDistance
	}
	if len(openResponse) > 0 {
		openDistance, _ := strconv.ParseFloat(openResponse, 64)
		settings["open_distance"] = openDistance
	}
	geofence["type"] = "circular"
	geofence["settings"] = settings

	return geofence
}

func runTeslamateGeofencePrompts() interface{} {
	fmt.Println("We will now work on configuring your teslamate geofence for this garage door.")
	geofence := map[string]interface{}{}

	settings := map[string]interface{}{}

	closeFromResponse := promptUser(question{
		prompt:             "Which teslamate geofence should you be leaving to trigger a garage close event (leave blank to skip automatically closing garage)? []",
		validResponseRegex: ".*",
	})
	var closeToResponse string
	if len(closeFromResponse) > 0 {
		closeToResponse = promptUser(question{
			prompt:                 "Which teslamate geofence should you be entering to trigger a garage close event?",
			validResponseRegex:     ".+",
			invalidResponseMessage: "Please enter a teslamate geofence name",
		})
		settings["close_trigger"] = map[string]string{
			"from": closeFromResponse,
			"to":   closeToResponse,
		}
	}
	openFromResponse := promptUser(question{
		prompt:             "Which teslamate geofence should you be leaving to trigger a garage open event (leave blank to skip automatically opening garage)? []",
		validResponseRegex: ".*",
	})
	var openToResponse string
	if len(openFromResponse) > 0 {
		openToResponse = promptUser(question{
			prompt:                 "Which teslamate geofence should you be entering to trigger a garage open event?",
			validResponseRegex:     ".+",
			invalidResponseMessage: "Please enter a teslamate geofence name",
		})
		settings["open_trigger"] = map[string]string{
			"from": openFromResponse,
			"to":   openToResponse,
		}
	}

	geofence["type"] = "geofence"
	geofence["settings"] = settings

	fmt.Println("Configuration of teslamate geofence for this garage door is complete, moving on...")

	return geofence
}

func runPolygonGeofencePrompts() interface{} {
	fmt.Println("We will now work on configuring your polygon geofence for this garage door.")

	geofence := map[string]interface{}{}
	geofence["type"] = "polygon"
	settings := map[string]interface{}{}

	var kmlFileResponse string
	kmlResponse := promptUser(question{
		prompt:                 "Would you like to use a KML file to define your polygon geofence? Choose y to define the file path, choose n to manually input latitude and longitude coordinates: [y|n]",
		validResponseRegex:     "^(y|Y|n|N)$",
		invalidResponseMessage: "Please respond with y or n",
	})
	if match, _ := regexp.MatchString("y|Y", kmlResponse); match {
		kmlFileResponse = promptUser(question{
			prompt:             "Please enter the file path of your KML file. Remember that if you're running this in a docker container, the file path will need to be relative to the container's filesystem:",
			validResponseRegex: ".+",
		})
		settings["kml_file"] = kmlFileResponse
		geofence["settings"] = settings
		return geofence
	}

	var openCoords []map[string]interface{}
	var closeCoords []map[string]interface{}
	re := regexp.MustCompile("n|N") // used to check if we should keep adding points to a geofence

	// run loop twice, once for open, the other for close
	for i := 0; i <= 1; i++ {
		var action string
		if i == 0 {
			action = "open"
		} else {
			action = "close"
		}
		// run loop to add as many points as the user wishes
		for {
			latResponse := promptUser(question{
				prompt:                 fmt.Sprintf("Define a latitude point in your %s geofence: ", action),
				validResponseRegex:     "^(-)?\\d+\\.\\d+$",
				invalidResponseMessage: "Please enter a valid latitude, such as 123.456 or -123.456",
			})
			lngResponse := promptUser(question{
				prompt:                 fmt.Sprintf("Define a longitude point in your %s geofence: ", action),
				validResponseRegex:     "^(-)?\\d+\\.\\d+$",
				invalidResponseMessage: "Please enter a valid longitude, such as 123.456 or -123.456",
			})
			lat, _ := strconv.ParseFloat(latResponse, 64)
			lng, _ := strconv.ParseFloat(lngResponse, 64)
			coordinates := map[string]interface{}{
				"lat": lat,
				"lng": lng,
			}
			if action == "open" {
				openCoords = append(openCoords, coordinates)
			} else {
				closeCoords = append(closeCoords, coordinates)
			}
			continueResponse := promptUser(question{
				prompt:                 fmt.Sprintf("Would you like to add another point to your polygon %s geofence? [y|n]", action),
				validResponseRegex:     "^(y|Y|n|N)$",
				invalidResponseMessage: "Please enter y or n",
			})
			if match := re.MatchString(continueResponse); match {
				settings[action] = openCoords
				break //inner
			}
		}
	}

	if len(openCoords) > 0 {
		settings["open"] = openCoords
	}
	if len(closeCoords) > 0 {
		settings["close"] = closeCoords
	}
	geofence["settings"] = settings

	fmt.Println("Configuration of polygon geofence for this garage door is complete, moving on...")

	return geofence
}
