package homeassistant

import (
	"encoding/json"
	"os"
	"strings"

	httpGdo "github.com/brchri/tesla-geogdo/internal/gdo/http"
	"github.com/brchri/tesla-geogdo/internal/util"
	logger "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// stubbed struct to extract api key and entity id from the yaml to pass into what is expected by httpGdo
type HomeAssistant struct {
	Settings struct {
		Connection struct {
			ApiKey string `yaml:"api_key"`
		} `yaml:"connection"`
		EntityId           string `yaml:"entity_id"`
		EnableStatusChecks bool   `yaml:"enable_status_checks"`
	} `yaml:"settings"`
}

func init() {
	logger.SetFormatter(&util.CustomFormatter{})
	logger.SetOutput(os.Stdout)
	if val, ok := os.LookupEnv("DEBUG"); ok && strings.ToLower(val) == "true" {
		logger.SetLevel(logger.DebugLevel)
	}
}

// this is just a wrapper for the http package with some predefined settings for homeassistant
func Initialize(config map[string]interface{}) (httpGdo.HttpGdo, error) {
	h, err := NewHomeAssistantGdo(config)
	if err != nil {
		return nil, err
	}
	return h, nil
}

func NewHomeAssistantGdo(config map[string]interface{}) (httpGdo.HttpGdo, error) {
	var hassGdo *HomeAssistant
	// marshall map[string]interface into yaml, then unmarshal to object based on yaml def in struct
	yamlData, err := yaml.Marshal(config)
	if err != nil {
		logger.Fatal("Failed to marhsal garage doors yaml object")
	}
	err = yaml.Unmarshal(yamlData, &hassGdo)
	if err != nil {
		logger.Fatal("Failed to unmarshal garage doors yaml object")
	}

	// add homeassistant-specific http settings to the config object
	if httpSettings, ok := config["settings"].(map[string]interface{}); ok {
		httpSettings["commands"] = []map[string]interface{}{
			{
				"name":                  "open",
				"endpoint":              "/api/services/cover/open_cover",
				"http_method":           "post",
				"body":                  `{"entity_id": "` + hassGdo.Settings.EntityId + `"}`,
				"required_start_state":  "closed",
				"required_finish_state": "open",
				"headers": []string{
					"Authorization: Bearer " + hassGdo.Settings.Connection.ApiKey,
					"Content-Type: application/json",
				},
			},
			{
				"name":                  "close",
				"endpoint":              "/api/services/cover/close_cover",
				"http_method":           "post",
				"body":                  `{"entity_id": "` + hassGdo.Settings.EntityId + `"}`,
				"required_start_state":  "open",
				"required_finish_state": "closed",
				"headers": []string{
					"Authorization: Bearer " + hassGdo.Settings.Connection.ApiKey,
					"Content-Type: application/json",
				},
			},
		}

		if hassGdo.Settings.EnableStatusChecks {
			httpSettings["status"] = map[string]interface{}{
				"endpoint": "/api/states/" + hassGdo.Settings.EntityId,
				"headers": []string{
					"Authorization: Bearer " + hassGdo.Settings.Connection.ApiKey,
					"Content-Type: application/json",
				},
			}
		}
	}

	// create new httpGdo object with updated config
	h, err := httpGdo.NewHttpGdo(config)
	if err != nil {
		return nil, err
	}

	// set callback function for httpGdo object to parse returned garage status
	h.SetParseStatusResponseFunc(ParseStatusResponse)

	return h, nil
}

// define a callback function for the httpGdo package to extract the garage status from the returned json
// all that's needed is the json value for the `state` key
func ParseStatusResponse(status string) (string, error) {
	type statusResponse struct {
		State string `json:"state"`
	}

	var s statusResponse
	err := json.Unmarshal([]byte(status), &s)
	if err != nil {
		logger.Debugf("Unable to parse")
	}

	return s.State, nil
}
