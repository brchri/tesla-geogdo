package console

import (
	"fmt"
	"strconv"
	"strings"

	asciiArt "github.com/common-nighthawk/go-figure"
)

func runHomeAssistantOpenerPrompts() interface{} {
	type (
		HassOpener struct {
			Type     string `yaml:"type"`
			Settings struct {
				Connection struct {
					Host          string `yaml:"host"`
					Port          int    `yaml:"port"`
					ApiKey        string `yaml:"api_key"`
					UseTls        bool   `yaml:"use_tls,omitempty"`
					SkipTlsVerify bool   `yaml:"skip_tls_verify,omitempty"`
				} `yaml:"connection"`
				EntityId           string `yaml:"entity_id"`
				EnableStatusChecks bool   `yaml:"enable_status_checks,omitempty"`
			} `yaml:"settings"`
		}
	)

	opener := HassOpener{
		Type: "homeassistant",
	}
	asciiArt.NewFigure("Home Assistant", "", false).Print()
	fmt.Print("\nWe will now configure your Home Assistant controlled garage door opener\n\n")

	opener.Settings.Connection.Host = promptUser(question{
		prompt:             "Please enter the DNS or IP for your Home Assistant server: ",
		validResponseRegex: ".+",
	})
	response := promptUser(question{
		prompt:                 "Please enter the port of your Home Assistant server [8123]: ",
		validResponseRegex:     "^\\d{1,5}$",
		invalidResponseMessage: "Please enter a valid port between 1 and 65535",
		defaultResponse:        "8123",
	})
	port, _ := strconv.Atoi(response)
	opener.Settings.Connection.Port = port
	opener.Settings.Connection.ApiKey = promptUser(question{
		prompt:             "Please enter your Home Assistant API key (see Home Assistant documentation for instructions on how to retrieve this): ",
		validResponseRegex: ".+",
	})
	response = promptUser(question{
		prompt:                 "Does your Home Assistant server require TLS? [y|N]",
		validResponseRegex:     yesNoRegexString,
		invalidResponseMessage: yesNoInvalidResponse,
		defaultResponse:        "n",
	})
	if isYesRegex.MatchString(response) {
		opener.Settings.Connection.UseTls = true
		response = promptUser(question{
			prompt:                 "Do you want to skip TLS verification (useful for self-signed certificates)? [y|N]",
			validResponseRegex:     yesNoRegexString,
			invalidResponseMessage: yesNoInvalidResponse,
			defaultResponse:        "n",
		})
		if isYesRegex.MatchString(response) {
			opener.Settings.Connection.SkipTlsVerify = true
		}
	}

	opener.Settings.EntityId = promptUser(question{
		prompt:             "Please enter the the entity ID for your garage door opener in Home Assistant (this can be found by adding '/config/entities' to the base URL for Home Assistant in your browser address bar): ",
		validResponseRegex: ".+",
	})

	response = promptUser(question{
		prompt:                 "Should status checks for the garage door be enabled? This will validate that, for example, a door is closed before trying to open it. [Y|n]",
		validResponseRegex:     yesNoRegexString,
		invalidResponseMessage: yesNoInvalidResponse,
		defaultResponse:        "y",
	})
	if isYesRegex.MatchString(response) {
		opener.Settings.EnableStatusChecks = true
	}

	fmt.Println("Your Home Assistant controlled garage door opener has been configured")

	return opener
}

func runHomebridgeOpenerPrompts() interface{} {
	type HomebridgeOpener struct {
		Type     string `yaml:"type"`
		Settings struct {
			Connection struct {
				Host          string `yaml:"host"`
				Port          int    `yaml:"port"`
				User          string `yaml:"user"`
				Pass          string `yaml:"pass"`
				UseTls        bool   `yaml:"use_tls,omitempty"`
				SkipTLSVerify bool   `yaml:"skip_tls_verify,omitempty"`
			} `yaml:"connection"`
			Timeout   int `yaml:"timeout"`
			Accessory struct {
				UniqueId        string `yaml:"unique_id"`
				Characteristics struct {
					Status  string `yaml:"status,omitempty"`
					Command string `yaml:"command"`
					Values  struct {
						Open  interface{} `yaml:"open"`
						Close interface{} `yaml:"close"`
					} `yaml:"values"`
				} `yaml:"characteristics"`
			} `yaml:"accessory"`
		} `yaml:"settings"`
	}

	asciiArt.NewFigure("Homebridge", "", false).Print()
	fmt.Print("\nWe will now configure your Homebridge controlled garage door opener\n\n")

	opener := HomebridgeOpener{
		Type: "homebridge",
	}

	opener.Settings.Connection.Host = promptUser(question{
		prompt:             "Please enter the DNS or IP of your Homebridge server: ",
		validResponseRegex: ".+",
	})
	response := promptUser(question{
		prompt:                 "Please enter the port of your Homebridge server: [8581]",
		validResponseRegex:     "^\\d{1,5}$",
		invalidResponseMessage: "Please enter a valid port between 1 and 65535",
		defaultResponse:        "8581",
	})
	port, _ := strconv.Atoi(response)
	opener.Settings.Connection.Port = port
	opener.Settings.Connection.User = promptUser(question{
		prompt:             "Please enter the username for your Homebridge server: ",
		validResponseRegex: ".+",
	})
	opener.Settings.Connection.Pass = promptUser(question{
		prompt:             "Please enter the password for your Homebridge server (NOTE: your characters will NOT be hidden! you can choose to enter this manually later if you wish)",
		validResponseRegex: ".*",
	})
	response = promptUser(question{
		prompt:                 "Does your Homebridge server require TLS? [y|N]",
		validResponseRegex:     yesNoRegexString,
		invalidResponseMessage: yesNoInvalidResponse,
		defaultResponse:        "n",
	})
	if isYesRegex.MatchString(response) {
		opener.Settings.Connection.UseTls = true
		response = promptUser(question{
			prompt:                 "Do you want to skip TLS verification (useful for self-signed certificates)? [y|N]",
			validResponseRegex:     yesNoRegexString,
			invalidResponseMessage: yesNoInvalidResponse,
			defaultResponse:        "n",
		})
		if isYesRegex.MatchString(response) {
			opener.Settings.Connection.SkipTLSVerify = true
		}
	}

	response = promptUser(question{
		prompt:                 "Please enter a timeout in seconds for the garage action to complete (how long should we wait for a door to open or close): [30]",
		validResponseRegex:     "^\\d+$",
		invalidResponseMessage: "Please enter a valid whole number",
		defaultResponse:        "30",
	})
	timeout, _ := strconv.Atoi(response)
	opener.Settings.Timeout = timeout

	opener.Settings.Accessory.UniqueId = promptUser(question{
		prompt:             "Please enter the unique ID for the garage door accessory in Homebridge (can be retrieved from the /swagger page of Homebridge with the /api/accessories endpoint)",
		validResponseRegex: ".+",
	})
	opener.Settings.Accessory.Characteristics.Status = promptUser(question{
		prompt:             "Please enter the status characteristic of the opener. This provides the status of the door, such as open or closed. This is optional. 'CurrentDoorState' is common. []",
		validResponseRegex: ".*",
	})
	opener.Settings.Accessory.Characteristics.Command = promptUser(question{
		prompt:             "Please enter the command characteristic of the opener. This defines how to control the door. [TargetDoorState]",
		validResponseRegex: ".+",
		defaultResponse:    "TargetDoorState",
	})
	response = promptUser(question{
		prompt:             "Please enter the value of the command characteristic that opens the door: [0]",
		validResponseRegex: ".+",
		defaultResponse:    "0",
	})
	openValue, err := strconv.Atoi(response)
	if err == nil {
		opener.Settings.Accessory.Characteristics.Values.Open = openValue
	} else {
		opener.Settings.Accessory.Characteristics.Values.Open = response
	}
	response = promptUser(question{
		prompt:             "Please enter the value of the command characteristic that closes the door: [1]",
		validResponseRegex: ".+",
		defaultResponse:    "1",
	})
	closeValue, err := strconv.Atoi(response)
	if err == nil {
		opener.Settings.Accessory.Characteristics.Values.Close = closeValue
	} else {
		opener.Settings.Accessory.Characteristics.Values.Close = response
	}

	fmt.Println("Your Homebridge controlled garage door opener has been configured")
	return opener
}

func runRatgdoOpenerPrompts() interface{} {
	type RatgdoOpener struct {
		Type         string `yaml:"type"`
		MqttSettings struct {
			Connection struct {
				Host          string `yaml:"host"`
				Port          int    `yaml:"port"`
				ClientId      string `yaml:"client_id,omitempty"`
				User          string `yaml:"user"`
				Pass          string `yaml:"pass"`
				UseTls        bool   `yaml:"use_tls,omitempty"`
				SkipTlsVerify bool   `yaml:"skip_tls_verify,omitempty"`
			} `yaml:"connection"`
			TopicPrefix string `yaml:"topic_prefix"`
		} `yaml:"mqtt_settings"`
	}

	asciiArt.NewFigure("ratgdo", "", false).Print()
	fmt.Print("\nWe will now configure your ratgdo opener for this door\n\n")
	opener := RatgdoOpener{
		Type: "ratgdo",
	}

	opener.MqttSettings.Connection.Host = promptUser(
		question{
			prompt:             "What is the DNS or IP of your ratgdo's MQTT broker? Note - if running in a container, you should not use localhost or 127.0.0.1",
			validResponseRegex: ".+",
		},
	)
	response := promptUser(
		question{
			prompt:                 "What is the port of your ratgdo's MQTT broker? [1883]",
			validResponseRegex:     "^\\d{1,5}$",
			invalidResponseMessage: "Please enter a number between 1 and 65534",
			defaultResponse:        "1883",
		},
	)
	port, _ := strconv.Atoi(response)
	opener.MqttSettings.Connection.Port = port
	opener.MqttSettings.Connection.ClientId = promptUser(
		question{
			prompt:             "Please enter the MQTT client ID to connect to the broker (can be left blank to auto generate at runtime); NOTE: THIS MUST BE UNIQUE FROM ANY OTHER CLIENTS CONNECTING TO THIS BROKER: []",
			validResponseRegex: ".*",
		},
	)
	opener.MqttSettings.Connection.User = promptUser(
		question{
			prompt:             "If your MQTT broker requires authentication, please enter the username: []",
			validResponseRegex: ".*",
		},
	)
	opener.MqttSettings.Connection.Pass = promptUser(
		question{
			prompt:             "If your MQTT broker requires authentication, please enter the password (your typing will not be masked!); you can also enter this later: []",
			validResponseRegex: ".*",
		},
	)
	response = promptUser(
		question{
			prompt:                 "Does your broker require TLS? [y|N]",
			validResponseRegex:     yesNoRegexString,
			invalidResponseMessage: yesNoInvalidResponse,
			defaultResponse:        "n",
		},
	)
	if isYesRegex.MatchString(response) {
		opener.MqttSettings.Connection.UseTls = true
		response = promptUser(
			question{
				prompt:                 "Do you want to skip TLS verification (useful for self-signed certificates)? [y|N]",
				validResponseRegex:     yesNoRegexString,
				invalidResponseMessage: yesNoInvalidResponse,
				defaultResponse:        "n",
			},
		)
		if isYesRegex.MatchString(response) {
			opener.MqttSettings.Connection.SkipTlsVerify = true
		}
	}

	opener.MqttSettings.TopicPrefix = promptUser(question{
		prompt:             "Please enter the topic prefix for your ratgdo (you can retrieve this from your ratgdo's settings page): ",
		validResponseRegex: ".+",
	})

	fmt.Print("You have finished configuring your ratgdo opener for this door\n\n")
	return opener
}

func runHttpOpenerPrompts() interface{} {
	type (
		Command struct {
			Name                string   `yaml:"name"`
			Endpoint            string   `yaml:"endpoint"`
			HttpMethod          string   `yaml:"http_method"`
			Body                string   `yaml:"body,omitempty"`
			RequiredStartState  string   `yaml:"required_start_state,omitempty"`
			RequiredFinishState string   `yaml:"required_finish_state,omitempty"`
			Timeout             int      `yaml:"timeout,omitempty"`
			Headers             []string `yaml:"headers,omitempty"`
		}

		HttpOpener struct {
			Type     string `yaml:"type"`
			Settings struct {
				Connection struct {
					Host          string `yaml:"host"`
					Port          int    `yaml:"port"`
					User          string `yaml:"user"`
					Pass          string `yaml:"pass"`
					UseTls        bool   `yaml:"use_tls,omitempty"`
					SkipTLSVerify bool   `yaml:"skip_tls_verify,omitempty"`
				} `yaml:"connection"`
				Status struct {
					Endpoint string   `yaml:"status,omitempty"`
					Headers  []string `yaml:"headers,omitempty"`
				} `yaml:"status,omitempty"`
				Commands []Command `yaml:"commands"`
			} `yaml:"settings"`
		}
	)

	asciiArt.NewFigure("Generic HTTP Opener", "", false).Print()
	fmt.Print("\nWe will now configure your generic HTTP opener for this door\n\n")
	opener := HttpOpener{
		Type: "http",
	}

	opener.Settings.Connection.Host = promptUser(question{
		prompt:             "Please enter the DNS or IP of the HTTP endpoint: ",
		validResponseRegex: ".+",
	})
	response := promptUser(question{
		prompt:                 "Please enter the port of the HTTP endpoint: [80]",
		validResponseRegex:     "^\\d{1,5}$",
		invalidResponseMessage: "Please enter a valid port between 1 and 65535",
		defaultResponse:        "80",
	})
	port, _ := strconv.Atoi(response)
	opener.Settings.Connection.Port = port
	response = promptUser(question{
		prompt:                 "Does your HTTP endpoint require basic authentication? [y|N]",
		validResponseRegex:     yesNoRegexString,
		invalidResponseMessage: yesNoInvalidResponse,
		defaultResponse:        "n",
	})
	if isYesRegex.MatchString(response) {
		opener.Settings.Connection.User = promptUser(question{
			prompt:             "Please enter the username for HTTP basic authentication for this endpoint: ",
			validResponseRegex: ".+",
		})
		opener.Settings.Connection.Pass = promptUser(question{
			prompt:             "Please enter the password for HTTP basic authentication for this endpoint. NOTE - YOUR TYPED CHARACTERS WILL NOT BE MASKED! You can add this to the config directly later if you wish: []",
			validResponseRegex: ".*",
		})
	}
	response = promptUser(question{
		prompt:                 "Does your endpoint use TLS (HTTPS)? [y|N]",
		validResponseRegex:     yesNoRegexString,
		invalidResponseMessage: yesNoInvalidResponse,
		defaultResponse:        "n",
	})
	if isYesRegex.MatchString(response) {
		opener.Settings.Connection.UseTls = true
		response = promptUser(question{
			prompt:                 "Do you want to skip TLS verification (useful for self-signed certificates)? [y|N]",
			validResponseRegex:     yesNoRegexString,
			invalidResponseMessage: yesNoInvalidResponse,
			defaultResponse:        "n",
		})
		if isYesRegex.MatchString(response) {
			opener.Settings.Connection.SkipTLSVerify = true
		}
	}

	response = promptUser(question{
		prompt:                 "Would you like to configure a status endpoint? This is used to validate garage state before taking an action (e.g. garage must be closed to send open command): [y|n]",
		validResponseRegex:     "^(y|Y|n|N)",
		invalidResponseMessage: yesNoInvalidResponse,
	})
	if isYesRegex.MatchString(response) {
		opener.Settings.Status.Endpoint = promptUser(question{
			prompt:             "Please enter the endpoint to retrieve the status from the HTTP server (e.g. /status): ",
			validResponseRegex: ".+",
		})
		opener.Settings.Status.Headers = getHttpHeadersHelperPrompts()
	}

	// run twice, once for open and another for close
	action := "open"
	for i := 0; i < 2; i++ {
		command := Command{
			Name: action,
		}
		command.Endpoint = promptUser(question{
			prompt:             fmt.Sprintf("Please enter the endpoint used to %s your garage (e.g. /command): ", strings.ToUpper(action)),
			validResponseRegex: ".+",
		})
		command.HttpMethod = promptUser(question{
			prompt:                 "Please enter the HTTP method used for the endpoint (e.g. GET, POST, PUT, etc)",
			validResponseRegex:     "^([Gg][Ee][Tt]|[Pp][Uu][Tt]|[Pp][Oo][Ss][Tt]|[Pp][Aa][Tt][Cc][Hh])$",
			invalidResponseMessage: "Please use one of the following methods: GET, PUT, POST, PATCH",
		})
		command.Body = promptUser(question{
			prompt:             fmt.Sprintf("Please enter the body of the request that should be submitted to the endpoint to %s the door (e.g. '{ \"command\": \"%s\" }'); leave blank if no body is required: []", strings.ToUpper(action), action),
			validResponseRegex: ".*",
		})
		if opener.Settings.Status.Endpoint != "" { // this block is only applicable if we can get the status of the door from an endpoint
			command.RequiredStartState = promptUser(question{
				prompt:             fmt.Sprintf("Entire the required start state to %s the garage (e.g. must be 'closed' to open, or must be open to close). Leave blank to disable prerequisite checks: []", strings.ToUpper(action)),
				validResponseRegex: ".*",
			})
			command.RequiredFinishState = promptUser(question{
				prompt:             fmt.Sprintf("Entire the required finish state to %s the garage (e.g. must be 'closed' when closing to be considered complete). Leave blank to disable check: []", strings.ToUpper(action)),
				validResponseRegex: ".*",
			})
			response = promptUser(question{
				prompt:             fmt.Sprintf("Please enter the timeout (in seconds) to wait for the door to finish %s command. Leave blank to disable: []", strings.ToUpper(action)),
				validResponseRegex: "^(\\d+)?$",
			})
			if len(response) > 0 {
				timeout, _ := strconv.Atoi(response)
				command.Timeout = timeout
			}
		}
		command.Headers = getHttpHeadersHelperPrompts()

		opener.Settings.Commands = append(opener.Settings.Commands, command)
		action = "close"
	}

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

	asciiArt.NewFigure("Generic MQTT Opener", "", false).Print()
	fmt.Print("We will now work on configuring your Generic MQTT Opener\n\n")

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
