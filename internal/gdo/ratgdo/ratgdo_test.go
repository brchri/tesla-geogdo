package ratgdo

import (
	"path/filepath"
	"testing"

	"github.com/brchri/tesla-geogdo/internal/util"
	"github.com/stretchr/testify/assert"
)

var sampleYaml = map[string]interface{}{
	"mqtt_settings": map[string]interface{}{
		"connection": map[string]interface{}{
			"host":            "localhost",
			"port":            1883,
			"client_id":       "test-mqtt-module",
			"user":            "test-user",
			"pass":            "test-pass",
			"use_tls":         false,
			"skip_tls_verify": false,
		},
		"topic_prefix": "home/garage/Main",
	},
}

// Since ratgdo is just a wrapper for mqttGdo with some predefined configs,
// just need to ensure NewRatgdo doesn't throw any errors when returning
// an MqttGdo object
func Test_NewRatgdo(t *testing.T) {
	// test with sample config defined above
	_, err := NewRatgdo(sampleYaml)
	assert.Equal(t, nil, err)

	// test with sample config extracted from example config.yml file
	util.LoadConfig(filepath.Join("..", "..", "..", "examples", "config.circular.ratgdo.yml"))
	door := *util.Config.GarageDoors[0]
	var openerConfig interface{}
	for k, v := range door {
		if k == "opener" {
			openerConfig = v
		}
	}
	if openerConfig == nil {
		t.Error("unable to parse config from garage door")
		return
	}
	config, ok := openerConfig.(map[string]interface{})
	if !ok {
		t.Error("unable to parse config from garage door")
		return
	}
	_, err = NewRatgdo(config)
	assert.Equal(t, nil, err)
}
