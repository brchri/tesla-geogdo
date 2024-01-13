package console

import (
	"fmt"
	"strconv"

	"github.com/brchri/tesla-geogdo/internal/geo"
	asciiArt "github.com/common-nighthawk/go-figure"
)

func runCircularGeofencePrompts() interface{} {
	asciiArt.NewFigure("Circular Geofence", "", false).Print()

	type CircularGeofence struct {
		Type     string               `yaml:"type"`
		Settings geo.CircularGeofence `yaml:"settings"`
	}
	fmt.Print("\nWe will now work on configuring your circular geofence for this garage door.\n\n")
	geofence := CircularGeofence{
		Type: "circular",
	}

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

	geofence.Settings.Center.Lat, _ = strconv.ParseFloat(latResponse, 64)
	geofence.Settings.Center.Lng, _ = strconv.ParseFloat(lngResponse, 64)

	if len(closeResponse) > 0 {
		closeDistance, _ := strconv.ParseFloat(closeResponse, 64)
		geofence.Settings.CloseDistance = closeDistance
	}
	if len(openResponse) > 0 {
		openDistance, _ := strconv.ParseFloat(openResponse, 64)
		geofence.Settings.OpenDistance = openDistance
	}

	return geofence
}

func runTeslamateGeofencePrompts() interface{} {
	asciiArt.NewFigure("Teslamate Geofence", "", false).Print()

	type TeslamateGeofence struct {
		Type     string                `yaml:"type"`
		Settings geo.TeslamateGeofence `yaml:"settings"`
	}

	fmt.Print("\nWe will now work on configuring your teslamate geofence for this garage door.\n\n")
	geofence := TeslamateGeofence{
		Type: "geofence",
	}

	geofence.Settings.Close.From = promptUser(question{
		prompt:             "Which teslamate geofence should you be leaving to trigger a garage close event (leave blank to skip automatically closing garage)? []",
		validResponseRegex: ".*",
	})
	if len(geofence.Settings.Close.From) > 0 {
		geofence.Settings.Close.To = promptUser(question{
			prompt:                 "Which teslamate geofence should you be entering to trigger a garage close event?",
			validResponseRegex:     ".+",
			invalidResponseMessage: "Please enter a teslamate geofence name",
		})
	}
	geofence.Settings.Open.From = promptUser(question{
		prompt:             "Which teslamate geofence should you be leaving to trigger a garage open event (leave blank to skip automatically opening garage)? []",
		validResponseRegex: ".*",
	})
	if len(geofence.Settings.Open.From) > 0 {
		geofence.Settings.Open.To = promptUser(question{
			prompt:                 "Which teslamate geofence should you be entering to trigger a garage open event?",
			validResponseRegex:     ".+",
			invalidResponseMessage: "Please enter a teslamate geofence name",
		})
	}

	fmt.Println("Configuration of teslamate geofence for this garage door is complete, moving on...")

	return geofence
}

func runPolygonGeofencePrompts() interface{} {
	asciiArt.NewFigure("Polygon Geofence", "", false).Print()

	type PolygonGeofence struct {
		Type     string              `yaml:"type"`
		Settings geo.PolygonGeofence `yaml:"settings"`
	}
	fmt.Print("\nWe will now work on configuring your polygon geofence for this garage door.\n\n")

	geofence := PolygonGeofence{
		Type: "polygon",
	}

	kmlResponse := promptUser(question{
		prompt:                 "Would you like to use a KML file to define your polygon geofence? Choose y to define the file path, choose n to manually input latitude and longitude coordinates: [y|n]",
		validResponseRegex:     yesNoRegexString,
		invalidResponseMessage: yesNoInvalidResponse,
	})
	if isYesRegex.MatchString(kmlResponse) {
		geofence.Settings.KMLFile = promptUser(question{
			prompt:             "Please enter the file path of your KML file. Remember that if you're running this in a docker container, the file path will need to be relative to the container's filesystem:",
			validResponseRegex: ".+",
		})
		fmt.Println("Configuration of polygon geofence for this garage door is complete, moving on...")
		return geofence
	}

	// run loop twice, once for open, the other for close
	var action string
	for i := 0; i <= 1; i++ {
		if i == 0 {
			action = "open"
		} else {
			action = "close"
		}
		// run loop to add as many points as the user wishes
		response := promptUser(question{
			prompt:                 fmt.Sprintf("Would you like to configure coordinate points for a geofence of type %s? [Y|n]: ", action),
			validResponseRegex:     yesNoRegexString,
			invalidResponseMessage: yesNoInvalidResponse,
			defaultResponse:        "y",
		})
		if isNoRegex.MatchString(response) {
			continue
		}
		for {
			p := geo.Point{}
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
			p.Lat, _ = strconv.ParseFloat(latResponse, 64)
			p.Lng, _ = strconv.ParseFloat(lngResponse, 64)
			if action == "open" {
				geofence.Settings.Open = append(geofence.Settings.Open, p)
			} else {
				geofence.Settings.Close = append(geofence.Settings.Close, p)
			}
			response := promptUser(question{
				prompt:                 fmt.Sprintf("Would you like to add another point to your polygon %s geofence? [y|n]", action),
				validResponseRegex:     "^(y|Y|n|N)$",
				invalidResponseMessage: "Please enter y or n",
			})
			if isNoRegex.MatchString(response) {
				break //inner
			}
		}
	}

	fmt.Println("Configuration of polygon geofence for this garage door is complete, moving on...")

	return geofence
}
