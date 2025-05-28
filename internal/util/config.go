package util

import (
	"fmt"
	"os"
	"strings"

	logger "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type (
	ConfigStruct struct {
		Global struct {
			MqttSettings struct {
				Connection MqttConnectSettings `yaml:"connection"`
			} `yaml:"tracker_mqtt_settings"`
			OpCooldown int `yaml:"cooldown"`
		} `yaml:"global"`
		GarageDoors  []*map[string]interface{} `yaml:"garage_doors"` // this will be parsed properly later by the geo package
		Testing      bool
		MasterOpLock int // user-initiated lock (pause), values: 0 = disabled, <0 = indefinite pause, >0 = num seconds left on finite timeout
	}

	MqttConnectSettings struct {
		Host          string `yaml:"host"`
		Port          int    `yaml:"port"`
		ClientID      string `yaml:"client_id"`
		User          string `yaml:"user"`
		Pass          string `yaml:"pass"`
		UseTls        bool   `yaml:"use_tls"`
		SkipTlsVerify bool   `yaml:"skip_tls_verify"`
	}

	CustomFormatter struct {
		logger.TextFormatter
	}
)

var Config ConfigStruct

func init() {
	logger.SetFormatter(&CustomFormatter{})
	logger.SetOutput(os.Stdout)
	if val, ok := os.LookupEnv("DEBUG"); ok && strings.ToLower(val) == "true" {
		logger.SetLevel(logger.DebugLevel)
	}
}

// format log level to always have 5 characters between brackets (e.g. `[INFO ]`)
func formatLevel(level logger.Level) string {
	displayMap := map[string]string{
		"INFO":    "INF",
		"WARNING": "WRN",
		"DEBUG":   "DBG",
		"ERROR":   "ERR",
		"TRACE":   "TRC",
		"FATAL":   "FTL",
	}

	displayLevel, ok := displayMap[strings.ToUpper(fmt.Sprintf("%v", level))]
	if !ok {
		displayLevel = strings.ToUpper(fmt.Sprintf("%v", level))
	}
	return " " + displayLevel + " "
}

// custom formatter for logrus package
// 01/02/2006 15:04:05 [LEVEL] Message...
func (f *CustomFormatter) Format(entry *logger.Entry) ([]byte, error) {
	// Use the timestamp from the log entry to format it as you like
	var timestamp string
	if os.Getenv("DEBUG") == "true" {
		// include milliseconds for debug
		timestamp = entry.Time.Format("01/02/2006 15:04:05.000")
	} else {
		timestamp = entry.Time.Format("01/02/2006 15:04:05")
	}

	// Ensure the log level string is always 5 characters
	paddedLevel := formatLevel(entry.Level)

	// Combine the timestamp with the log level and the message
	logMessage := fmt.Sprintf("%s [%s] %s\n", timestamp, paddedLevel, entry.Message)
	return []byte(logMessage), nil
}

// load yaml config
func LoadConfig(configFile string) {
	logger.Debugf("Attempting to read config file: %v", configFile)
	yamlFile, err := os.ReadFile(configFile)
	if err != nil {
		logger.Fatalf("Could not read config file: %v", err)
	} else {
		logger.Debug("Config file read successfully")
	}

	logger.Debug("Unmarshaling yaml into config object")
	err = yaml.Unmarshal(yamlFile, &Config)
	if err != nil {
		logger.Fatalf("Could not load yaml from config file, received error: %v", err)
	} else {
		logger.Debug("Config yaml unmarshalled successfully")
	}

	logger.Info("Config loaded successfully")
}
