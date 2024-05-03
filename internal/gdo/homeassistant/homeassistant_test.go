package homeassistant

import (
	"path/filepath"
	"testing"

	"github.com/brchri/tesla-geogdo/internal/util"
	"github.com/stretchr/testify/assert"
)

var sampleYaml = map[string]interface{}{
	"settings": map[string]interface{}{
		"connection": map[string]interface{}{
			"host":            "localhost",
			"port":            80,
			"api_key":         "somelongtoken",
			"use_tls":         false,
			"skip_tls_verify": false,
		},
		"entity_id":            "cover.main_cover",
		"enable_status_checks": true,
	},
}

// Since homeassistant is just a wrapper for httpGdo with some predefined configs,
// just need to ensure NewRatgdo doesn't throw any errors when returning
// an httpGdo object
func Test_NewHomeAssistantGdo(t *testing.T) {
	// test with sample config defined above
	_, err := NewHomeAssistantGdo(sampleYaml)
	assert.Equal(t, nil, err)

	// test with sample config extracted from example config.yml file
	util.LoadConfig(filepath.Join("..", "..", "..", "examples", "config.circular.homeassistant.yml"))
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
	_, err = NewHomeAssistantGdo(config)
	assert.Equal(t, nil, err)
}
