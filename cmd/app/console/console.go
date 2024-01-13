package console

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	asciiArt "github.com/common-nighthawk/go-figure"
	"gopkg.in/yaml.v3"
)

type question struct {
	prompt                 string
	validResponseRegex     string
	invalidResponseMessage string
	defaultResponse        string
}

var reader = bufio.NewReader(os.Stdin)
var yesNoRegexString = "^(?i)(y|n)$"

var isYesRegex = regexp.MustCompile("y|Y")
var isNoRegex = regexp.MustCompile("n|N")
var yesNoInvalidResponse = "Please respond with y or n"

func RunWizard() {
	shutdownSignal := make(chan os.Signal, 1)
	// handle interrupts
	signal.Notify(shutdownSignal, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-shutdownSignal
		os.Exit(0)
	}()

	type Config struct {
		Global      interface{}   `yaml:"global"`
		GarageDoors []interface{} `yaml:"garage_doors"`
	}

	asciiArt.NewFigure("Tesla-GeoGDO", "", false).Print()
	asciiArt.NewFigure("Config Wizard", "", false).Print()

	// config := map[string]interface{}{}
	config := Config{}
	response := promptUser(
		question{
			prompt:                 "\n\nWould you like to use the wizard to generate your config file? [Y|n]",
			validResponseRegex:     yesNoRegexString,
			invalidResponseMessage: yesNoInvalidResponse,
			defaultResponse:        "y",
		},
	)
	if isNoRegex.MatchString(response) {
		return
	}

	displayConfigStructureInformation()
	config.Global = runGlobalPrompts()
	config.GarageDoors = runGarageDoorsPrompts()

	var b bytes.Buffer
	yamlEncoder := yaml.NewEncoder(&b)
	yamlEncoder.SetIndent(2)
	yamlEncoder.Encode(config)
	yamlString := b.String()
	fmt.Print("\n\n####################### CONFIG FILE #######################")
	fmt.Print("\n\n" + yamlString + "\n\n")
	fmt.Print("##################### END CONFIG FILE #####################\n\n")

	asciiArt.NewFigure("Config Wizard Complete", "", false)

	fmt.Println("Congratulations on completing the config wizard. You can view your generated config above. You can choose to save this file automatically now, or you can copy and paste the contents above into your config file location.")
	response = promptUser(question{
		prompt:                 "Save file now? [Y|n]",
		validResponseRegex:     yesNoRegexString,
		invalidResponseMessage: yesNoInvalidResponse,
		defaultResponse:        "y",
	})
	if isNoRegex.MatchString(response) {
		return
	}
	filePath := promptUser(question{
		prompt:             "Where should the config file be saved? (remember, if running in a container, file path must be relative to container's file system) [/app/config/config.yml]",
		validResponseRegex: ".*",
		defaultResponse:    "/app/config/config.yml",
	})
	err := os.WriteFile(filePath, b.Bytes(), 0644)
	if err != nil {
		fmt.Printf("ERROR: Unable to write file to %s", filePath)
		os.Exit(1)
	} else {
		fmt.Println("\nSave complete!")
	}
}

func runGlobalPrompts() interface{} {
	type Global struct {
		TrackerMqttSettings struct {
			Connection struct {
				Host          string `yaml:"host"`
				Port          int    `yaml:"port"`
				ClientId      string `yaml:"client_id,omitempty"`
				User          string `yaml:"user"`
				Pass          string `yaml:"pass"`
				UseTls        bool   `yaml:"use_tls,omitempty"`
				SkipTlsVerify bool   `yaml:"skip_tls_verify,omitempty"`
			} `yaml:"connection"`
		} `yaml:"tracker_mqtt_settings"`
		Cooldown int `yaml:"cooldown,omitempty"`
	}
	asciiArt.NewFigure("Global Config", "", false).Print()

	global := Global{}

	global.TrackerMqttSettings.Connection.Host = promptUser(
		question{
			prompt:             "\nWhat is the DNS or IP of your tracker's MQTT broker (e.g. teslamate)? Note - if running in a container, you should not use localhost or 127.0.0.1",
			validResponseRegex: ".+",
		},
	)
	response := promptUser(
		question{
			prompt:                 "What is the port of your tracker's MQTT broker (e.g. teslamate)? [1883]",
			validResponseRegex:     "^\\d{1,5}$",
			invalidResponseMessage: "Please enter a number between 1 and 65534",
			defaultResponse:        "1883",
		},
	)
	port, _ := strconv.Atoi(response)
	global.TrackerMqttSettings.Connection.Port = port
	global.TrackerMqttSettings.Connection.ClientId = promptUser(
		question{
			prompt:             "Please enter the MQTT client ID to connect to the broker (can be left blank to auto generate at runtime): []",
			validResponseRegex: ".*",
		},
	)
	response = promptUser(question{
		prompt:                 "Does your MQTT broker require authentication? [y|N]",
		validResponseRegex:     yesNoRegexString,
		invalidResponseMessage: yesNoInvalidResponse,
		defaultResponse:        "n",
	})
	if isYesRegex.MatchString(response) {
		global.TrackerMqttSettings.Connection.User = promptUser(
			question{
				prompt:             "Please enter the username for the tracker's MQTT broker (or press [Enter] to leave blank and manually enter it into the config file later): []",
				validResponseRegex: ".*",
			},
		)
		global.TrackerMqttSettings.Connection.Pass = promptUser(
			question{
				prompt:             "Please enter the password for the tracker's MQTT broker (or press [Enter] to leave blank and manually enter it into the config file later)\nWARNING: TYPED CHARACTERS WILL NOT BE MASKED: []",
				validResponseRegex: ".*",
			},
		)
	}
	response = promptUser(
		question{
			prompt:                 "Does your broker require TLS? [y|N]",
			validResponseRegex:     yesNoRegexString,
			invalidResponseMessage: yesNoInvalidResponse,
			defaultResponse:        "n",
		},
	)
	if isYesRegex.MatchString(response) {
		global.TrackerMqttSettings.Connection.UseTls = true
		response = promptUser(
			question{
				prompt:                 "Do you want to skip TLS verification (useful for self-signed certificates)? [y|N]",
				validResponseRegex:     yesNoRegexString,
				invalidResponseMessage: yesNoInvalidResponse,
				defaultResponse:        "n",
			},
		)
		if isYesRegex.MatchString(response) {
			global.TrackerMqttSettings.Connection.SkipTlsVerify = true
		}
	}

	response = promptUser(
		question{
			prompt:                 "Set the number (in minutes) that there should be a global cooldown for each door.\nThis prevents any door from being operated for a set time after any previous operation.\nLeave blank to disable global cooldown: []",
			validResponseRegex:     "^(\\d+)?$",
			invalidResponseMessage: "Please enter a valid number (in minutes)",
		},
	)
	if len(response) > 0 {
		cooldown, _ := strconv.Atoi(response)
		global.Cooldown = cooldown
	}
	return global
}

func runGarageDoorsPrompts() []interface{} {
	asciiArt.NewFigure("Garage Doors", "", false).Print()

	fmt.Print("\nWe will now configure one or more garage doors, which will include geofences, openers, and trackers\n\n")
	garage_doors := []interface{}{}
	re := regexp.MustCompile("n|N")

	for {
		garage_door := map[string]interface{}{}
		var response string
		response = promptUser(
			question{
				prompt:                 "What type of geofence would you like to configure for this garage door (more doors can be added later)?  [c|t|p]\nc: circular\nt: teslamate\np: polygon",
				validResponseRegex:     "^(?i)(c|t|p)$",
				invalidResponseMessage: "Please enter c (for circular), t (for teslamate), or p (for polygon)",
			},
		)
		switch response {
		case "c":
			garage_door["geofence"] = runCircularGeofencePrompts()
		case "t":
			garage_door["geofence"] = runTeslamateGeofencePrompts()
		case "p":
			garage_door["geofence"] = runPolygonGeofencePrompts()
		}

		response = promptUser(question{
			prompt:                 "\nWhat type of garage door opener would you like to configure for this garage door? [ha|hb|r|h|m]\nha: Home Assistant\nhb: Homebridge\nr: ratgdo (MQTT firmware only; for ESP Home, control via Home Assistant or Homebridge)\nh: Generic HTTP\nm: Generic MQTT",
			validResponseRegex:     "^(?i)(ha|hb|r|h|m)$",
			invalidResponseMessage: "Please enter ha (for Home Assistant), hb (for Homebridge), r (for ratgdo), h (for generic HTTP), or m (for generic MQTT)",
		})

		switch response {
		case "ha":
			garage_door["opener"] = runHomeAssistantOpenerPrompts()
		case "hb":
			garage_door["opener"] = runHomebridgeOpenerPrompts()
		case "r":
			garage_door["opener"] = runRatgdoOpenerPrompts()
		case "h":
			garage_door["opener"] = runHttpOpenerPrompts()
		case "m":
			garage_door["opener"] = runMqttOpenerPrompts()
		}

		garage_door["trackers"] = runTrackerPrompts()

		garage_doors = append(garage_doors, garage_door)

		asciiArt.NewFigure("Garage Door Complete", "", false).Print()

		response = promptUser(question{
			prompt:                 "\nWould you like to configure another garage door? [y|n]",
			validResponseRegex:     yesNoRegexString,
			invalidResponseMessage: yesNoInvalidResponse,
		})
		if re.MatchString(response) {
			break
		}
	}

	fmt.Println("Configuring garage doors is complete!")
	return garage_doors
}

func promptUser(q question) string {
	fmt.Println(q.prompt)
	response := readResponse()
	if len(response) == 0 && q.defaultResponse != "" {
		return q.defaultResponse
	}
	match, _ := regexp.MatchString(q.validResponseRegex, response)
	if !match {
		if q.invalidResponseMessage != "" {
			fmt.Println("-- " + q.invalidResponseMessage + " --")
		}
		return promptUser(q)
	}
	return response
}

func readResponse() string {
	text, _ := reader.ReadString('\n')
	text = strings.Replace(text, "\n", "", -1)
	text = strings.Replace(text, "\r", "", -1)
	return text
}

func displayConfigStructureInformation() {
	asciiArt.NewFigure("Config Structure", "", false).Print()
	fmt.Println("\nFirst we'll discuss the structure or anatomy of the config file")
	fmt.Println("\nThere are 2 main sections of the config file: global and garage_doors")
	fmt.Println("\nThe settings in the global section apply to all garage doors and trackers")
	fmt.Println("Here we define the MQTT broker where the tracker location updates are published")
	fmt.Println("(i.e. where Teslamate, Owntracks, or other trackers publish locations)")
	fmt.Println("as well as a cooldown that is applied to all garage doors, which prevents")
	fmt.Print("any door from being operated if it was already operated recently.\n\n")

	fmt.Println("The next section is garage_doors. Here's where you'll configure each garage door that will be controlled")
	fmt.Println("Each garage door will have the following definitions:")
	fmt.Println("  * Open and close geofences that indicate when to trigger garage door open and close events")
	fmt.Println("  * An opener, that tells the app how to operate your garage door")
	fmt.Println("  * One or more trackers - for example, in a 2-car garage, you might have 2 trackers defined, 1 for each vehicle")

	fmt.Println("\n\nFirst we'll configure the global section.")
	fmt.Println("Press [Enter] to continue when ready...")

	readResponse()
}
