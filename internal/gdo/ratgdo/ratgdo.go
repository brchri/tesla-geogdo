package ratgdo

import (
	"os"
	"strings"

	mqttGdo "github.com/brchri/tesla-geogdo/internal/gdo/mqtt"
	"github.com/brchri/tesla-geogdo/internal/util"
	logger "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// stubbed mqtt struct to extract the topic prefix from the yaml to pass into what is expected by mqttGdo
type Ratgdo struct {
	MqttSettings struct {
		TopicPrefix string `yaml:"topic_prefix"`
	} `yaml:"mqtt_settings"`
}

func init() {
	logger.SetFormatter(&util.CustomFormatter{})
	logger.SetOutput(os.Stdout)
	if val, ok := os.LookupEnv("DEBUG"); ok && strings.ToLower(val) == "true" {
		logger.SetLevel(logger.DebugLevel)
	}
}

// this is just a wrapper for the mqtt package with some predefined settings for ratgdo
func Initialize(config map[string]interface{}) (mqttGdo.MqttGdo, error) {
	m, err := NewRatgdo(config)
	if err != nil {
		return nil, err
	}
	m.InitializeMqttClient()
	return m, nil
}

func NewRatgdo(config map[string]interface{}) (mqttGdo.MqttGdo, error) {
	var ratgdo *Ratgdo
	// marshall map[string]interface into yaml, then unmarshal to object based on yaml def in struct
	yamlData, err := yaml.Marshal(config)
	if err != nil {
		logger.Fatal("Failed to marshal garage doors yaml object")
	}
	err = yaml.Unmarshal(yamlData, &ratgdo)
	if err != nil {
		logger.Fatal("Failed to unmarshal garage doors yaml object")
	}

	// add ratgdo-specific mqtt settings to the config object
	config["settings"] = config["mqtt_settings"] // mqtt expects just `settings` key
	if mqttSettings, ok := config["settings"].(map[string]interface{}); ok {
		mqttSettings["topics"] = map[string]string{
			"prefix":       ratgdo.MqttSettings.TopicPrefix,
			"door_status":  "status/door",
			"obstruction":  "status/obstruction",
			"availability": "status/availability",
		}
		mqttSettings["commands"] = []map[string]string{
			{
				"name":                  "open",
				"payload":               "open",
				"topic_suffix":          "command/door",
				"required_start_state":  "closed",
				"required_finish_state": "open",
			}, {
				"name":                  "close",
				"payload":               "close",
				"topic_suffix":          "command/door",
				"required_start_state":  "open",
				"required_finish_state": "closed",
			},
		}
	}

	return mqttGdo.NewMqttGdo(config)
}
