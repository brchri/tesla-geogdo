package console

import (
	"regexp"
	"strconv"
)

func runHomeAssistantOpenerPrompts() interface{} {
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

	return opener
}
