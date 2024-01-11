package console

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type question struct {
	prompt                 string
	validResponseRegex     string
	invalidResponseMessage string
	defaultResponse        string
}

var reader = bufio.NewReader(os.Stdin)

func RunWizard() {
	config := map[string]interface{}{}
	response := promptUser(
		question{
			prompt:                 "Would you like to use the wizard to generate your config file? [Y|n]",
			validResponseRegex:     "^(y|Y|n|N)$",
			invalidResponseMessage: "Please respond with y or n",
			defaultResponse:        "y",
		},
	)
	if match, _ := regexp.MatchString("n|N|No|no|NO", response); match {
		return
	}

	// config["global"] = runGlobalPrompts()
	config["garage_doors"] = runGarageDoorsPrompts()

	yamlData, _ := yaml.Marshal(config)
	yamlString := string(yamlData)
	fmt.Print("\n\n" + yamlString)
}

func runGlobalPrompts() interface{} {
	response := promptUser(
		question{
			prompt:             "What is the DNS or IP of your tracker's MQTT broker (e.g. teslamate)? Note - if running in a container, you should not use localhost or 127.0.0.1",
			validResponseRegex: ".+",
		},
	)
	tracker_connection := map[string]interface{}{
		"host": response,
	}
	response = promptUser(
		question{
			prompt:                 "What is the port of your tracker's MQTT broker (e.g. teslamate)? [1883]",
			validResponseRegex:     "^\\d{1,5}$",
			invalidResponseMessage: "Please enter a number between 1 and 65534",
			defaultResponse:        "1883",
		},
	)
	port, _ := strconv.ParseFloat(response, 64)
	tracker_connection["port"] = port
	response = promptUser(
		question{
			prompt:             "Please enter the MQTT client ID to connect to the broker (can be left blank to auto generate at runtime): []",
			validResponseRegex: ".*",
		},
	)
	if response != "" {
		tracker_connection["client_id"] = response
	}
	response = promptUser(
		question{
			prompt:             "If your MQTT broker requires authentication, please enter the username: []",
			validResponseRegex: ".*",
		},
	)
	if response != "" {
		tracker_connection["user"] = response
	}
	response = promptUser(
		question{
			prompt:             "If your MQTT broker requires authentication, please enter the password (your typing will not be masked!); you can also enter this later: []",
			validResponseRegex: ".*",
		},
	)
	if response != "" {
		tracker_connection["pass"] = response
	}
	response = promptUser(
		question{
			prompt:                 "Does your broker require TLS? [y|N]",
			validResponseRegex:     "^(y|Y|n|N)$",
			invalidResponseMessage: "Please respond with y or n",
			defaultResponse:        "n",
		},
	)
	if match, _ := regexp.MatchString("y|Y", response); match {
		tracker_connection["use_tls"] = true
		response = promptUser(
			question{
				prompt:                 "Do you want to skip TLS verification (useful for self-signed certificates)? [y|N]",
				validResponseRegex:     "^(y|Y|n|N)$",
				invalidResponseMessage: "Please respond with y or n",
				defaultResponse:        "n",
			},
		)
		if match, _ := regexp.MatchString("y|Y", response); match {
			tracker_connection["skip_tls_verify"] = true
		}
	}

	tracker_mqtt_settings := map[string]interface{}{
		"tracker_mqtt_settings": map[string]interface{}{
			"connection": tracker_connection,
		},
	}

	global_config := map[string]interface{}{
		"tracker_mqtt_settings": tracker_mqtt_settings,
	}

	response = promptUser(
		question{
			prompt:                 "Set the number (in minutes) that there should be a global cooldown for each door. This prevents any door from being operated for a set time after any previous operation. []",
			validResponseRegex:     "^(\\d+)?$",
			invalidResponseMessage: "Please enter a valid number (in minutes)",
		},
	)
	if len(response) > 0 {
		cooldown, _ := strconv.Atoi(response)
		global_config["cooldown"] = cooldown
	}
	return global_config
}

func runGarageDoorsPrompts() []interface{} {
	garage_doors := []interface{}{}
	garage_door := map[string]interface{}{}
	var response string
	response = promptUser(
		question{
			prompt:                 "What type of geofence would you like to configure for this garage door (more doors can be added later)?  [c|t|p]\nc: circular\nt: teslamate\np: polygon",
			validResponseRegex:     "^(c|t|p|C|T|P)$",
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
		prompt:                 "What type of garage door opener would you like to configure for this garage door? [ha|hb|r|h|m]\nha: Home Assistant\nhb: Homebridge\nr: ratgdo\nh: Generic HTTP\nm: Generic MQTT",
		validResponseRegex:     "^(ha|hb|r|h|m)$",
		invalidResponseMessage: "Please enter ha (for Home Assistant), hb (for Homebridge), r (for ratgdo), h (for generic HTTP), or m (for generic MQTT)",
	})

	switch response {
	case "ha":
		garage_door["opener"] = runHomeAssistantOpenerPrompts()
	}

	garage_doors = append(garage_doors, garage_door)

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
		fmt.Println(q.invalidResponseMessage)
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
