package mqtt

import (
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/brchri/tesla-geogdo/internal/mocks"
	"github.com/brchri/tesla-geogdo/internal/util"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var sampleYaml = map[string]interface{}{
	"settings": map[string]interface{}{
		"connection": map[string]interface{}{
			"host":            "localhost",
			"port":            1883,
			"client_id":       "test-mqtt-module",
			"user":            "test-user",
			"pass":            "test-pass",
			"use_tls":         false,
			"skip_tls_verify": false,
		},
		"topics": map[string]interface{}{
			"prefix":       "home/garage/Main",
			"door_status":  "status/door",
			"obstruction":  "status/obstruction",
			"availability": "status/availability",
		},
		"commands": []map[string]interface{}{
			{
				"name":                  "open",
				"payload":               "open",
				"topic_suffix":          "command/door",
				"required_start_state":  "closed",
				"required_finish_state": "open",
				"timeout":               5,
			}, {
				"name":                  "close",
				"payload":               "close",
				"topic_suffix":          "command/door",
				"required_start_state":  "open",
				"required_finish_state": "closed",
				"timeout":               5,
			},
		},
	},
}

func Test_NewClient(t *testing.T) {
	// test with sample config defined above
	mqttgdo, err := NewMqttGdo(sampleYaml)
	assert.Equal(t, nil, err)
	if err != nil {
		return
	}

	if m, ok := mqttgdo.(*mqttGdo); ok {
		assert.Equal(t, m.Settings.Connection.Host, "localhost")
		assert.Equal(t, m.Settings.Connection.Port, 1883)
		assert.Equal(t, m.Settings.Topics.DoorStatus, "status/door")
		assert.Equal(t, m.Settings.Commands[0].Name, "open")
		assert.Equal(t, m.Settings.Commands[1].Timeout, 5)
	} else {
		t.Error("returned type is not *mqttGdo")
	}

	// test with sample config extracted from example config.yml file
	util.LoadConfig(filepath.Join("..", "..", "..", "examples", "config.teslamate.mqtt.yml"))
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
	_, err = NewMqttGdo(config)
	assert.Equal(t, nil, err)
}

func Test_InitializeClient(t *testing.T) {
	// initialize mock objects
	mockMqttClient := &mocks.Client{}
	mockMqttClient.Test(t)
	mockMqttToken := &mocks.Token{}
	mockMqttToken.Test(t)
	defer mockMqttClient.AssertExpectations(t)
	defer mockMqttToken.AssertExpectations(t)

	// set expectations for assertion
	mockMqttToken.EXPECT().Wait().Once().Return(true)
	mockMqttToken.EXPECT().Error().Once().Return(nil)
	mockMqttClient.EXPECT().Connect().Once().Return(mockMqttToken)
	mockMqttClient.EXPECT().IsConnected().Once().Return(true)

	// override mqtt.NewClient function with mocked function
	mqttNewClientFunc = func(o *mqtt.ClientOptions) mqtt.Client { return mockMqttClient }

	// initialize test object
	mqttGdo := &mqttGdo{}

	mqttGdo.InitializeMqttClient()
}

func Test_SetGarageDoor_WithStatus(t *testing.T) {
	// initialize mock objects
	mockMqttClient := &mocks.Client{}
	mockMqttClient.Test(t)
	mockMqttToken := &mocks.Token{}
	mockMqttToken.Test(t)
	defer mockMqttClient.AssertExpectations(t)
	defer mockMqttToken.AssertExpectations(t)

	// set expectations for assertion
	mockMqttToken.EXPECT().Wait().Once().Return(true)
	mockMqttClient.EXPECT().Publish("home/garage/Main/command/door", mock.Anything, false, "open").Once().Return(mockMqttToken)

	// initialize test object
	m, err := NewMqttGdo(sampleYaml)
	assert.Equal(t, nil, err)
	if err != nil {
		return
	}
	mqttGdo, ok := m.(*mqttGdo) // check type so we can access structs
	if !ok {
		t.Error("returned type is not *mqttGdo")
	}
	mqttGdo.State = "closed"
	mqttGdo.MqttClient = mockMqttClient

	// run in go routine so we can set the door status after making the call so the function doesn't wait for the timeout
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		assert.Equal(t, nil, mqttGdo.SetGarageDoor("open"))
	}()

	// wait for SetGarageDoor to call Publish with 5 second timeout
	start := time.Now()
	for mqttGdo.State != "open" && time.Since(start) < 5*time.Second {
		for _, v := range mockMqttClient.Calls {
			if v.Method == "Publish" {
				mqttGdo.State = "open"
			}
		}
	}

	wg.Wait()
}

func Test_SetGarageDoor_NoStatus(t *testing.T) {
	// initialize mock objects
	mockMqttClient := &mocks.Client{}
	mockMqttClient.Test(t)
	mockMqttToken := &mocks.Token{}
	mockMqttToken.Test(t)
	defer mockMqttClient.AssertExpectations(t)
	defer mockMqttToken.AssertExpectations(t)

	// set expectations for assertion
	mockMqttToken.EXPECT().Wait().Once().Return(true)
	mockMqttClient.EXPECT().Publish("home/garage/Main/command/door", mock.Anything, false, "open").Once().Return(mockMqttToken)

	// initialize test object
	m, err := NewMqttGdo(sampleYaml)
	assert.Equal(t, nil, err)
	if err != nil {
		return
	}
	mqttGdo, ok := m.(*mqttGdo) // check type so we can access structs
	if !ok {
		t.Error("returned type is not *mqttGdo")
	}
	mqttGdo.State = "closed"
	mqttGdo.MqttClient = mockMqttClient
	mqttGdo.Settings.Topics.DoorStatus = ""

	assert.Equal(t, nil, mqttGdo.SetGarageDoor("open"))
}
