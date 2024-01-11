package console

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

func runHomeAssistantOpenerPrompts() interface{} {
	fmt.Println("We will now configure your Home Assistant controlled garage door opener")
	opener := map[string]interface{}{}
	opener["type"] = "homeassistant"
	settings := map[string]interface{}{}
	connection := map[string]interface{}{}

	hostResponse := promptUser(question{
		prompt:             "Please enter the DNS or IP for your Home Assistant server: ",
		validResponseRegex: ".+",
	})
	portResponse := promptUser(question{
		prompt:                 "Please enter the port of your Home Assistant server [8123]: ",
		validResponseRegex:     "^\\d{1,5}$",
		invalidResponseMessage: "Please enter a valid port between 1 and 65535",
		defaultResponse:        "8123",
	})
	apiKeyResponse := promptUser(question{
		prompt:             "Please enter your Home Assistant API key (see Home Assistant documentation for instructions on how to retrieve this): ",
		validResponseRegex: ".+",
	})
	tlsResponse := promptUser(question{
		prompt:                 "Does your Home Assistant server require TLS? [y|N]",
		validResponseRegex:     "^(y|Y|n|N)$",
		invalidResponseMessage: "Please respond with y or n",
		defaultResponse:        "n",
	})
	if match, _ := regexp.MatchString("y|Y", tlsResponse); match {
		connection["use_tls"] = true
		skipTlsVerifyResponse := promptUser(question{
			prompt:                 "Do you want to skip TLS verification (useful for self-signed certificates)? [y|N]",
			validResponseRegex:     "^(y|Y|n|N)$",
			invalidResponseMessage: "Please respond with y or n",
			defaultResponse:        "n",
		})
		if match, _ := regexp.MatchString("n|N", skipTlsVerifyResponse); match {
			connection["skip_tls_verify"] = true
		}
	}

	connection["host"] = hostResponse
	port, _ := strconv.ParseFloat(portResponse, 64)
	connection["port"] = port
	connection["api_key"] = apiKeyResponse
	settings["connection"] = connection

	response := promptUser(question{
		prompt:             "Please enter the the entity ID for your garage door opener in Home Assistant (this can be found by adding '/config/entities' to the base URL for Home Assistant in your browser address bar): ",
		validResponseRegex: ".+",
	})
	settings["entity_id"] = response

	response = promptUser(question{
		prompt:                 "Should status checks for the garage door be enabled? This will validate that, for example, a door is closed before trying to open it. [Y|n]",
		validResponseRegex:     "^(y|Y|n|N)$",
		invalidResponseMessage: "Please respond with y or n",
		defaultResponse:        "y",
	})
	if match, _ := regexp.MatchString("y|Y", response); match {
		settings["enable_status_checks"] = true
	}

	opener["settings"] = settings

	fmt.Println("Your Home Assistant controlled garage door opener has been configured")

	return opener
}

func runHomebridgeOpenerPrompts() interface{} {
	fmt.Println("We will now configure your Homebridge controlled garage door opener")
	opener := map[string]interface{}{}
	opener["type"] = "homebridge"
	settings := map[string]interface{}{}
	connection := map[string]interface{}{}
	response := promptUser(question{
		prompt:             "Please enter the DNS or IP of your Homebridge server: ",
		validResponseRegex: ".+",
	})
	connection["host"] = response
	response = promptUser(question{
		prompt:                 "Please enter the port of your Homebridge server: [8581]",
		validResponseRegex:     "^\\d{1,5}$",
		invalidResponseMessage: "Please enter a valid port between 1 and 65535",
		defaultResponse:        "8581",
	})
	port, _ := strconv.ParseFloat(response, 64)
	connection["port"] = port
	response = promptUser(question{
		prompt:             "Please enter the username for your Homebridge server: ",
		validResponseRegex: ".+",
	})
	connection["user"] = response
	response = promptUser(question{
		prompt:             "Please enter the password for your Homebridge server (NOTE: your characters will NOT be hidden! you can choose to enter this manually later if you wish)",
		validResponseRegex: ".*",
	})
	connection["pass"] = response
	response = promptUser(question{
		prompt:                 "Does your Homebridge server require TLS? [y|N]",
		validResponseRegex:     "^(y|Y|n|N)$",
		invalidResponseMessage: "Please respond with y or n",
		defaultResponse:        "n",
	})
	if match, _ := regexp.MatchString("Y|y", response); match {
		connection["use_tls"] = true
		response = promptUser(question{
			prompt:                 "Do you want to skip TLS verification (useful for self-signed certificates)? [y|N]",
			validResponseRegex:     "^(y|Y|n|N)$",
			invalidResponseMessage: "Please respond with y or n",
			defaultResponse:        "n",
		})
		if match, _ := regexp.MatchString("y|Y", response); match {
			connection["skip_tls_verify"] = true
		}
	}
	settings["connection"] = connection

	response = promptUser(question{
		prompt:                 "Please enter a timeout in seconds for the garage action to complete (how long should we wait for a door to open or close): [30]",
		validResponseRegex:     "^\\d+$",
		invalidResponseMessage: "Please enter a valid whole number",
		defaultResponse:        "30",
	})
	timeout, _ := strconv.ParseFloat(response, 64)
	settings["timeout"] = timeout

	accessory := map[string]interface{}{}
	response = promptUser(question{
		prompt:             "Please enter the unique ID for the garage door accessory in Homebridge (can be retrieved from the /swagger page of Homebridge with the /api/accessories endpoint)",
		validResponseRegex: ".+",
	})
	accessory["unique_id"] = response
	characteristics := map[string]interface{}{}
	response = promptUser(question{
		prompt:             "Please enter the status characteristic of the opener. This provides the status of the door, such as open or closed. This is optional. 'CurrentDoorState' is common. []",
		validResponseRegex: ".*",
	})
	characteristics["status"] = response
	response = promptUser(question{
		prompt:             "Please enter the command characteristic of the opener. This defines how to control the door. [TargetDoorState]",
		validResponseRegex: ".+",
		defaultResponse:    "TargetDoorState",
	})
	characteristics["command"] = response
	values := map[string]interface{}{}
	response = promptUser(question{
		prompt:             "Please enter the value of the command characteristic that opens the door: [0]",
		validResponseRegex: ".+",
		defaultResponse:    "0",
	})
	values["open"] = response
	response = promptUser(question{
		prompt:             "Please enter the value of the command characteristic that closes the door: [1]",
		validResponseRegex: ".+",
		defaultResponse:    "1",
	})
	values["close"] = response

	characteristics["values"] = values
	accessory["characteristics"] = characteristics
	settings["accessory"] = accessory

	opener["settings"] = settings
	fmt.Println("Your Homebridge controlled garage door opener has been configured")
	return opener
}

func runRatgdoOpenerPrompts() interface{} {
	fmt.Print("We will now configure your ratgdo opener for this door\n\n")
	opener := map[string]interface{}{}
	opener["type"] = "ratgdo"
	mqtt_settings := map[string]interface{}{}
	connection := map[string]interface{}{}

	response := promptUser(
		question{
			prompt:             "What is the DNS or IP of your ratgdo's MQTT broker? Note - if running in a container, you should not use localhost or 127.0.0.1",
			validResponseRegex: ".+",
		},
	)
	connection["host"] = response
	response = promptUser(
		question{
			prompt:                 "What is the port of your ratgdo's MQTT broker? [1883]",
			validResponseRegex:     "^\\d{1,5}$",
			invalidResponseMessage: "Please enter a number between 1 and 65534",
			defaultResponse:        "1883",
		},
	)
	port, _ := strconv.ParseFloat(response, 64)
	connection["port"] = port
	response = promptUser(
		question{
			prompt:             "Please enter the MQTT client ID to connect to the broker (can be left blank to auto generate at runtime); NOTE: THIS MUST BE UNIQUE FROM ANY OTHER CLIENTS CONNECTING TO THIS BROKER: []",
			validResponseRegex: ".*",
		},
	)
	if response != "" {
		connection["client_id"] = response
	}
	response = promptUser(
		question{
			prompt:             "If your MQTT broker requires authentication, please enter the username: []",
			validResponseRegex: ".*",
		},
	)
	if response != "" {
		connection["user"] = response
	}
	response = promptUser(
		question{
			prompt:             "If your MQTT broker requires authentication, please enter the password (your typing will not be masked!); you can also enter this later: []",
			validResponseRegex: ".*",
		},
	)
	if response != "" {
		connection["pass"] = response
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
		connection["use_tls"] = true
		response = promptUser(
			question{
				prompt:                 "Do you want to skip TLS verification (useful for self-signed certificates)? [y|N]",
				validResponseRegex:     "^(y|Y|n|N)$",
				invalidResponseMessage: "Please respond with y or n",
				defaultResponse:        "n",
			},
		)
		if match, _ := regexp.MatchString("y|Y", response); match {
			connection["skip_tls_verify"] = true
		}
	}

	mqtt_settings["connection"] = connection

	response = promptUser(question{
		prompt:             "Please enter the topic prefix for your ratgdo (you can retrieve this from your ratgdo's settings page): ",
		validResponseRegex: ".+",
	})
	mqtt_settings["topic_prefix"] = response

	opener["mqtt_settings"] = mqtt_settings
	fmt.Print("You have finished configuring your ratgdo opener for this door\n\n")
	return opener
}

func runHttpOpenerPrompts() interface{} {
	fmt.Print("We will now configure your generic HTTP opener for this door\n\n")
	opener := map[string]interface{}{}
	opener["type"] = "http"
	settings := map[string]interface{}{}
	connection := map[string]interface{}{}

	response := promptUser(question{
		prompt:             "Please enter the DNS or IP of the HTTP endpoint: ",
		validResponseRegex: ".+",
	})
	connection["host"] = response
	response = promptUser(question{
		prompt:                 "Please enter the port of the HTTP endpoint: [80]",
		validResponseRegex:     "^\\d{1,5}$",
		invalidResponseMessage: "Please enter a valid port between 1 and 65535",
		defaultResponse:        "80",
	})
	port, _ := strconv.Atoi(response)
	connection["port"] = port
	response = promptUser(question{
		prompt:                 "Does your endpoint use TLS (HTTPS)? [y|N]",
		validResponseRegex:     "^(y|Y|n|N)$",
		invalidResponseMessage: "Please respond with y or n",
		defaultResponse:        "n",
	})
	if match, _ := regexp.MatchString("y|Y", response); match {
		connection["use_tls"] = true
		response = promptUser(question{
			prompt:                 "Do you want to skip TLS verification (useful for self-signed certificates)? [y|N]",
			validResponseRegex:     "^(y|Y|n|N)$",
			invalidResponseMessage: "Please respond with y or n",
			defaultResponse:        "n",
		})
		if match, _ := regexp.MatchString("y|Y", response); match {
			connection["skip_tls_verify"] = true
		}
	}
	response = promptUser(question{
		prompt:                 "Does your HTTP endpoint require basic authentication? [y|N]",
		validResponseRegex:     "^(y|Y|n|N)$",
		invalidResponseMessage: "Please respond with y or n",
		defaultResponse:        "n",
	})
	if match, _ := regexp.MatchString("y|Y", response); match {
		response = promptUser(question{
			prompt:             "Please enter the username for HTTP basic authentication for this endpoint: ",
			validResponseRegex: ".+",
		})
		connection["user"] = response
		response = promptUser(question{
			prompt:             "Please enter the password for HTTP basic authentication for this endpoint. NOTE - YOUR TYPED CHARACTERS WILL NOT BE MASKED! You can add this to the config directly later if you wish: []",
			validResponseRegex: ".*",
		})
		connection["pass"] = response
	}
	settings["connection"] = connection

	statusEndpointDefined := false
	response = promptUser(question{
		prompt:                 "Would you like to configure a status endpoint? This is used to validate garage state before taking an action (e.g. garage must be closed to send open command): [y|n]",
		validResponseRegex:     "^(y|Y|n|N)",
		invalidResponseMessage: "Please respond with y or n",
	})
	if match, _ := regexp.MatchString("y|Y", response); match {
		status := map[string]interface{}{}

		response = promptUser(question{
			prompt:             "Please enter the endpoint to retrieve the status from the HTTP server (e.g. /status): ",
			validResponseRegex: ".+",
		})
		status["endpoint"] = response
		headers := getHttpHeadersHelperPrompts()
		if len(headers) > 0 {
			status["headers"] = headers
		}

		settings["status"] = status
		statusEndpointDefined = true
	}

	commands := []map[string]interface{}{}
	// run twice, once for open and another for close
	action := "open"
	for i := 0; i < 2; i++ {
		command := map[string]interface{}{}
		command["name"] = action
		response = promptUser(question{
			prompt:             fmt.Sprintf("Please enter the endpoint used to %s your garage (e.g. /command): ", strings.ToUpper(action)),
			validResponseRegex: ".+",
		})
		command["endpoint"] = response
		response = promptUser(question{
			prompt:                 "Please enter the HTTP method used for the endpoint (e.g. GET, POST, PUT, etc)",
			validResponseRegex:     "^([Gg][Ee][Tt]|[Pp][Uu][Tt]|[Pp][Oo][Ss][Tt]|[Pp][Aa][Tt][Cc][Hh])$",
			invalidResponseMessage: "Please use one of the following methods: GET, PUT, POST, PATCH",
		})
		command["http_method"] = strings.ToLower(response)
		response = promptUser(question{
			prompt:             fmt.Sprintf("Please enter the body of the request that should be submitted to the endpoint to %s the door (e.g. '{ \"command\": \"%s\" }'); leave blank if no body is required: []", strings.ToUpper(action), action),
			validResponseRegex: ".*",
		})
		if len(response) > 0 {
			command["body"] = response
		}
		if statusEndpointDefined {
			response = promptUser(question{
				prompt:             fmt.Sprintf("Entire the required start state to %s the garage (e.g. must be 'closed' to open, or must be open to close). Leave blank to disable prerequisite checks: []", strings.ToUpper(action)),
				validResponseRegex: ".*",
			})
			if len(response) > 0 {
				command["required_start_state"] = response
			}
			response = promptUser(question{
				prompt:             fmt.Sprintf("Entire the required finish state to %s the garage (e.g. must be 'closed' when closing to be considered complete). Leave blank to disable check: []", strings.ToUpper(action)),
				validResponseRegex: ".*",
			})
			if len(response) > 0 {
				command["required_finish_state"] = response
			}
			response = promptUser(question{
				prompt:             fmt.Sprintf("Please enter the timeout (in seconds) to wait for the door to finish %s command. Leave blank to disable: []", strings.ToUpper(action)),
				validResponseRegex: "^(\\d+)?$",
			})
			if len(response) > 0 {
				timeout, _ := strconv.Atoi(response)
				command["timeout"] = timeout
			}
		}
		headers := getHttpHeadersHelperPrompts()
		if len(headers) > 0 {
			command["headers"] = headers
		}

		commands = append(commands, command)
		action = "close"
	}
	settings["commands"] = commands
	opener["settings"] = settings

	fmt.Print("You have finished configuring your generic HTTP opener for this door\n\n")
	return opener
}

func getHttpHeadersHelperPrompts() []string {
	response := promptUser(question{
		prompt:                 "Would you like to add any custom headers? [y|n]",
		validResponseRegex:     yesNoRegexString,
		invalidResponseMessage: yesNoInvalidResponse,
	})
	if isYesRegex.MatchString(response) {
		headers := []string{}
		for {
			response = promptUser(question{
				prompt:             "Please enter the header in key/pair format (e.g. 'Content-Type: application/json')",
				validResponseRegex: ".+",
			})
			headers = append(headers, response)
			response = promptUser(question{
				prompt:                 "Would you like to add another header? [y|n]",
				validResponseRegex:     yesNoRegexString,
				invalidResponseMessage: yesNoInvalidResponse,
			})
			if isNoRegex.MatchString(response) {
				return headers
			}
		}
	}
	return []string{}
}

func runMqttOpenerPrompts() interface{} {
	type (
		Command struct {
			Name                string `yaml:"name"`
			Payload             string `yaml:"payload"`
			TopicSuffix         string `yaml:"topic_suffix"`
			RequiredStartState  string `yaml:"required_start_state,omitempty"`
			RequiredFinishState string `yaml:"required_finish_state,omitempty"`
		}

		mqttOpener struct {
			Type     string `yaml:"type"`
			Settings struct {
				Connection struct {
					Host          string `yaml:"host"`
					Port          int    `yaml:"port"`
					User          string `yaml:"user,omitempty"`
					Pass          string `yaml:"pass,omitempty"`
					UseTls        bool   `yaml:"use_tls,omitempty"`
					SkipTlsVerify bool   `yaml:"skip_tls_verify,omitempty"`
					ClientId      bool   `yaml:"client_id,omitempty"`
				} `yaml:"connection"`
				Topics struct {
					Prefix       string `yaml:"prefix,omitempty"`
					DoorStatus   string `yaml:"door_status,omitempty"`
					Obstruction  string `yaml:"obstruction,omitempty"`
					Availability string `yaml:"availability,omitempty"`
				} `yaml:"topics,omitempty"`
				Commands []Command `yaml:"commands"`
			} `yaml:"settings"`
		}
	)

	opener := mqttOpener{
		Type: "mqtt",
	}
	response := promptUser(question{
		prompt:             "Please enter the DNS or IP for your opener's MQTT broker: ",
		validResponseRegex: ".+",
	})
	opener.Settings.Connection.Host = response
	response = promptUser(question{
		prompt:                 "Please enter the port of your opener's MQTT broker: [1883]",
		validResponseRegex:     "^\\d{1,5}$",
		invalidResponseMessage: "Please enter a valid port between 1 and 65535",
		defaultResponse:        "1883",
	})
	port, _ := strconv.Atoi(response)
	opener.Settings.Connection.Port = port
	response = promptUser(question{
		prompt:             "If your MQTT broker requires authentication, enter a username here (leave blank to disable): []",
		validResponseRegex: ".*",
	})
	opener.Settings.Connection.User = response
	response = promptUser(question{
		prompt:             "If your MQTT broker requires authentication, enter a password here (leave blank to disable); NOTE - typed characters WILL NOT be masked; you can enter this in the config file manually later if desired: []",
		validResponseRegex: ".*",
	})
	opener.Settings.Connection.Pass = response
	response = promptUser(question{
		prompt:                 "Does your MQTT broker require TLS? [y|N]",
		validResponseRegex:     yesNoRegexString,
		invalidResponseMessage: yesNoInvalidResponse,
		defaultResponse:        "n",
	})
	if isYesRegex.MatchString(response) {
		opener.Settings.Connection.UseTls = true
		response = promptUser(question{
			prompt:                 "Would you like to skip TLS verification (useful for self-signed certificates)? [y|N]",
			validResponseRegex:     yesNoRegexString,
			invalidResponseMessage: yesNoInvalidResponse,
			defaultResponse:        "n",
		})
		if isYesRegex.MatchString(response) {
			opener.Settings.Connection.SkipTlsVerify = true
		}
	}

	//topics
	response = promptUser(question{
		prompt:             "Please enter your topic prefix. All other prefixes defined will prepend this prefix to those topics. Leave blank if you want to define full topics for all entries later. []",
		validResponseRegex: ".*",
	})
	opener.Settings.Topics.Prefix = response
	opener.Settings.Topics.DoorStatus = promptUser(question{
		prompt:             "Enter the topic to get door status information. Remember if you defined a topic prefix, DO NOT include that in the topic name here. Leave blank to disable. []",
		validResponseRegex: ".*",
	})
	opener.Settings.Topics.Obstruction = promptUser(question{
		prompt:             "Enter the topic to get door obstruction information. Remember if you defined a topic prefix, DO NOT include that in the topic name here. Leave blank to disable. []",
		validResponseRegex: ".*",
	})
	opener.Settings.Topics.Availability = promptUser(question{
		prompt:             "Enter the topic to get door opener availability. Remember if you defined a topic prefix, DO NOT include that in the topic name here. Leave blank to disable. []",
		validResponseRegex: ".*",
	})

	//commands
	action := "open"
	for i := 0; i < 2; i++ {
		command := Command{
			Name: action,
		}
		command.Payload = promptUser(question{
			prompt:             fmt.Sprintf("Enter the payload to publish to the topic to %s the door", strings.ToUpper(action)),
			validResponseRegex: ".+",
		})
		command.TopicSuffix = promptUser(question{
			prompt:             fmt.Sprintf("Enter the topic suffix (or full topic if topic prefix was not defined earlier) to send the payload to %s the door", strings.ToUpper(action)),
			validResponseRegex: ".+",
		})
		command.RequiredStartState = promptUser(question{
			prompt:             fmt.Sprintf("Please enter the required start state to %s the garage (e.g. must be 'closed' to open). Leave blank to skip. []", strings.ToUpper(action)),
			validResponseRegex: ".*",
		})
		command.RequiredFinishState = promptUser(question{
			prompt:             fmt.Sprintf("Please enter the required state to consider the %s action complete (e.g. must be 'closed' to be considered closed). Leave blank to skip. []", strings.ToUpper(action)),
			validResponseRegex: ".*",
		})
		opener.Settings.Commands = append(opener.Settings.Commands, command)
		action = "close"
	}

	return opener
}
