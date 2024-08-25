package mqtt

import (
	"crypto/tls"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/brchri/tesla-geogdo/internal/util"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	logger "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type (
	// MqttGdo is the interface definition for an MqttGdo used by this library
	// The interface is primarily used for mocking tests
	MqttGdo interface {
		// InitializeMqttClient initializes the client by setting the connection options, connecting
		// to the mqtt broker, and subscribing to topics
		InitializeMqttClient()
		// onMqttConnect is the function handler that executes whenever the client connects (or reconnects)
		// to the mqtt broker; it primarly handles topic subscriptions
		onMqttConnect(mqtt.Client)
		// processMqttMessage receives the published messages on subscribed topics and updates the object properties accordingly
		processMqttMessage(mqtt.Client, mqtt.Message)
		// SetGarageDoor operates the garage door by publishing to the configured mqtt topic with the configured payload
		SetGarageDoor(string) error
		// process any required shutdown events, such as service disconnects
		ProcessShutdown()
	}

	// mqttGdo is the struct that implements the MqttGdo interface
	mqttGdo struct {
		Settings struct {
			Connection struct {
				Host          string `yaml:"host"`
				Port          int    `yaml:"port"`
				ClientID      string `yaml:"client_id"`
				User          string `yaml:"user"`
				Pass          string `yaml:"pass"`
				UseTls        bool   `yaml:"use_tls"`
				SkipTlsVerify bool   `yaml:"skip_tls_verify"`
			} `yaml:"connection"`
			Topics struct {
				Prefix       string `yaml:"prefix"` // prefixed to all subscription and command topics; can be blank if all other topics are fully defined
				DoorStatus   string `yaml:"door_status"`
				Obstruction  string `yaml:"obstruction"`
				Availability string `yaml:"availability"`
			} `yaml:"topics"`
			Commands []Command `yaml:"commands"`
		} `yaml:"settings"`
		OpenerType   string      `yaml:"type"` // name used by this module can be overridden by consuming modules, such as ratgdo, which is a wrapper for this package
		MqttClient   mqtt.Client // client that manages the connections and subscriptions to the mqtt broker
		State        string      // state of the garage door
		Availability string      // if the garage door controller publishes an availability status (e.g. online), it will be stored here
		Obstruction  string      // if the garage door controller publishes obstruction information, it will be stored here
	}

	Command struct {
		Name                string `yaml:"name"`                  // e.g. `open` or `close`
		Payload             string `yaml:"payload"`               // this could be the same or different than Name depending on the mqtt implementation
		TopicSuffix         string `yaml:"topic_suffix"`          // location where the command will be published; prefixed by MqttSettings.Topics.Prefix
		RequiredStartState  string `yaml:"required_start_state"`  // if set, garage door will not operate if current state does not equal this
		RequiredFinishState string `yaml:"required_finish_state"` // if set, garage door will monitor the door state compared to this value to determine success
		Timeout             int    `yaml:"timeout"`               // time to wait for garage door to operate if monitored
	}
)

const (
	defaultModuleName = "Generic MQTT Opener"
	defaultMqttPort   = 1883
)

var mqttNewClientFunc = mqtt.NewClient // abstract NewClient function call to allow mocking

func init() {
	logger.SetFormatter(&util.CustomFormatter{})
	logger.SetOutput(os.Stdout)
	if val, ok := os.LookupEnv("DEBUG"); ok && strings.ToLower(val) == "true" {
		logger.SetLevel(logger.DebugLevel)
	}
}

// wrapper function to parse the config, initialize the connection to the mqtt broker, and return the MqttGdo object
func Initialize(config map[string]interface{}) (MqttGdo, error) {
	mqttGdo, err := NewMqttGdo(config)
	if err != nil {
		return nil, err
	}
	mqttGdo.InitializeMqttClient()
	return mqttGdo, nil
}

// parses the config and returns an MqttGdo object
func NewMqttGdo(config map[string]interface{}) (MqttGdo, error) {
	var mqttGdo *mqttGdo
	// marshall map[string]interface into yaml, then unmarshal to object based on yaml def in struct
	yamlData, err := yaml.Marshal(config)
	if err != nil {
		logger.Fatal("Failed to marhsal garage doors yaml object")
	}
	err = yaml.Unmarshal(yamlData, &mqttGdo)
	if err != nil {
		logger.Fatal("Failed to unmarhsal garage doors yaml object")
	}

	if mqttGdo.OpenerType == "" {
		mqttGdo.OpenerType = defaultModuleName
	}

	// check if garage door opener is connecting to the same mqtt broker as the global for teslamate, and if so, that they have unique clientIDs
	localMqtt := &mqttGdo.Settings.Connection
	globalMqtt := util.Config.Global.MqttSettings.Connection
	if localMqtt.ClientID != "" && localMqtt.ClientID == globalMqtt.ClientID && localMqtt.Host == globalMqtt.Host && localMqtt.Port == globalMqtt.Port {
		localMqtt.ClientID = localMqtt.ClientID + "-" + mqttGdo.OpenerType + "-" + uuid.NewString()
		logger.Warnf("mqtt client id for door opener is the same as the global, appending opener type and random uuid to the client id: %s", localMqtt.ClientID)
	}

	// set command timeouts if not defined
	for k, v := range mqttGdo.Settings.Commands {
		if v.Timeout == 0 {
			mqttGdo.Settings.Commands[k].Timeout = 30
		}
	}

	mqttGdo.Settings.Topics.Prefix = strings.TrimRight(mqttGdo.Settings.Topics.Prefix, "/") // trim any trailing `/` on the prefix topic
	return mqttGdo, mqttGdo.ValidateMinimumMqttSettings()
}

// will validate that the minimum mqtt settings are defined,
// will return nil if all settings validated successfully or
// an error if not
// also populates missing required settings with assumed defaults,
// such as port=1883
//
// validated settings are:
// host, commands[*].{Name,Payload,TopicSuffix}
func (m *mqttGdo) ValidateMinimumMqttSettings() error {
	var errors []string
	if m.Settings.Connection.Host == "" {
		errors = append(errors, "missing mqtt host setting")
	}
	if len(m.Settings.Commands) == 0 {
		errors = append(errors, "at least 1 command required to operate garage")
	}
	for i, c := range m.Settings.Commands {
		commandErrorFormat := "missing %s for command %d"
		if c.Name == "" {
			errors = append(errors, fmt.Sprintf(commandErrorFormat, "command name", i))
		}
		if c.Payload == "" {
			errors = append(errors, fmt.Sprintf(commandErrorFormat, "command payload", i))
		}
		if c.TopicSuffix == "" {
			errors = append(errors, fmt.Sprintf(commandErrorFormat, "command topic suffix", i))
		}
	}

	// set defaults if omitted from config
	if m.Settings.Connection.Port == 0 {
		logger.Debugf("Port is undefined for %s, using default port %d", m.OpenerType, defaultMqttPort)
		m.Settings.Connection.Port = defaultMqttPort
	}

	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, "; "))
	}
	return nil
}

// sets mqtt client options and connects to the broker
func (m *mqttGdo) InitializeMqttClient() {

	logger.Debug("Setting MqttGdo MQTT Opts:")
	// create a new MQTT client
	opts := mqtt.NewClientOptions()
	logger.Debug(" OrderMatters: false")
	opts.SetOrderMatters(false)
	logger.Debug(" KeepAlive: 30 seconds")
	opts.SetKeepAlive(30 * time.Second)
	logger.Debug(" PingTimeout: 10 seconds")
	opts.SetPingTimeout(10 * time.Second)
	logger.Debug(" AutoReconnect: true")
	opts.SetAutoReconnect(true)
	if m.Settings.Connection.User != "" {
		logger.Debug(" Username: true <redacted value>")
	} else {
		logger.Debug(" Username: false (not set)")
	}
	opts.SetUsername(m.Settings.Connection.User) // if not defined, will just set empty strings and won't be used by pkg
	if m.Settings.Connection.Pass != "" {
		logger.Debug(" Password: true <redacted value>")
	} else {
		logger.Debug(" Password: false (not set)")
	}
	opts.SetPassword(m.Settings.Connection.Pass) // if not defined, will just set empty strings and won't be used by pkg
	opts.OnConnect = m.onMqttConnect

	// set conditional MQTT client opts
	if m.Settings.Connection.ClientID != "" {
		logger.Debugf(" ClientID: %s", m.Settings.Connection.ClientID)
		opts.SetClientID(m.Settings.Connection.ClientID)
	} else {
		// generate UUID for mqtt client connection if not specified in config file
		id := uuid.New().String()
		logger.Debugf(" ClientID: %s", id)
		opts.SetClientID(id)
	}
	logger.Debug(" Protocol: TCP")
	mqttProtocol := "tcp"
	if m.Settings.Connection.UseTls {
		logger.Debug(" UseTLS: true")
		logger.Debugf(" SkipTLSVerify: %t", m.Settings.Connection.SkipTlsVerify)
		opts.SetTLSConfig(&tls.Config{
			InsecureSkipVerify: m.Settings.Connection.SkipTlsVerify,
		})
		mqttProtocol = "ssl"
	} else {
		logger.Debug(" UseTLS: false")
	}
	broker := fmt.Sprintf("%s://%s:%d", mqttProtocol, m.Settings.Connection.Host, m.Settings.Connection.Port)
	logger.Debugf(" Broker: %s", broker)
	opts.AddBroker(broker)

	// create a new MQTT client object
	m.MqttClient = mqttNewClientFunc(opts)

	// connect to the MQTT broker
	logger.Debug("Connecting to MqttGdo MQTT broker")
	if token := m.MqttClient.Connect(); token.Wait() && token.Error() != nil {
		logger.Fatalf("%s could not connect to mqtt broker: %v", m.OpenerType, token.Error())
	} else {
		logger.Infof("%s door opener connected to MQTT broker", m.OpenerType)
	}
	logger.Debugf("MQTT Broker Connected: %t", m.MqttClient.IsConnected())
}

// function handler for when the mqtt client (re-)connects to the broker
// subscribes to topics if relevant
func (m *mqttGdo) onMqttConnect(client mqtt.Client) {
	topicSuffixes := []string{
		m.Settings.Topics.Obstruction,
		m.Settings.Topics.Availability,
		m.Settings.Topics.DoorStatus,
	}

	for _, t := range topicSuffixes {
		if t == "" {
			// don't process if any of the suffixes are empty
			continue
		}

		fullTopic := m.Settings.Topics.Prefix + "/" + t
		logger.Debugf("Subscribing to MqttGdo MQTT topic %s", fullTopic)
		topicSubscribed := false
		// retry topic subscription attempts with 1 sec delay between attempts
		for retryAttempts := 5; retryAttempts > 0; retryAttempts-- {
			logger.Debugf("Subscribing to topic: %s", fullTopic)
			if token := client.Subscribe(
				fullTopic,
				0,
				m.processMqttMessage); token.Wait() && token.Error() == nil {
				topicSubscribed = true
				logger.Debugf("Topic subscribed successfully: %s", fullTopic)
				break
			} else {
				logger.Infof("Failed to subscribe to topic %s for car mqttGdo, will make %d more attempts. Error: %v", fullTopic, retryAttempts, token.Error())
			}
			time.Sleep(5 * time.Second)
		}
		if !topicSubscribed {
			logger.Fatalf("Unable to subscribe to topics, exiting")
		}
	}
	logger.Debug("MqttGdo topics subscribed, listening for events...")
}

// handler to process messages published to subscribed topics
// sets mqttGdo properties based on payloads
func (m *mqttGdo) processMqttMessage(client mqtt.Client, message mqtt.Message) {
	// update MqttGdo property based on topic suffix (strip shared prefix on the switch)
	logger.Debugf("Received message on topic %s with payload %s", message.Topic(), string(message.Payload()))
	switch strings.TrimPrefix(message.Topic(), m.Settings.Topics.Prefix+"/") {
	case m.Settings.Topics.DoorStatus:
		logger.Debugf("Setting door status: %s", string(message.Payload()))
		m.State = string(message.Payload())
		logger.Debugf("Door status now set to: %s", m.State)
	case m.Settings.Topics.Availability:
		m.Availability = string(message.Payload())
	case m.Settings.Topics.Obstruction:
		m.Obstruction = string(message.Payload())
	default:
		logger.Debugf("invalid message topic: %s", message.Topic())
	}
}

// operates the garage door based on the supplied action by publishing
// the relevant payload to the configured command topic
// if configured, will monitor door status to confirm successful operation
func (m *mqttGdo) SetGarageDoor(action string) (err error) {
	var command Command
	for _, v := range m.Settings.Commands {
		if action == v.Name {
			command = v
			break
		}
	}

	if command.Name == "" {
		return fmt.Errorf("no command defined for action %s", action)
	}

	// if status topic and required state are defined, check that the required state is satisfied
	if m.Settings.Topics.DoorStatus != "" && command.RequiredStartState != "" && m.State != command.RequiredStartState {
		logger.Warnf("Action and state mismatch: garage state is not valid for executing requested action; current state: %s; requested action: %s", m.State, action)
		return
	}

	if util.Config.Testing {
		logger.Infof("TESTING flag set - Would attempt action %v", action)
		return
	}

	logger.Infof("setting garage door %s", action)
	logger.Debugf("Reported MqttGdo availability: %s", m.Availability)

	token := m.MqttClient.Publish(m.Settings.Topics.Prefix+"/"+command.TopicSuffix, 0, false, command.Payload)
	token.Wait()

	// if a required finish state and status topic are defined, wait for it to be satisfied
	if command.RequiredFinishState != "" && m.Settings.Topics.DoorStatus != "" {
		// wait for timeout
		start := time.Now()
		for time.Since(start) < time.Duration(command.Timeout)*time.Second {
			if m.State == command.RequiredFinishState {
				logger.Infof("Garage door state has been set successfully: %s", command.RequiredFinishState)
				return
			}
			logger.Debugf("Current opener state: %s", m.State)
			time.Sleep(1 * time.Second)
		}

		// these are based on ratgdo implementation, should probably make them configurable as other implementations may not use the same statuses
		if m.Settings.Topics.Availability != "" && m.Availability == "offline" {
			err = fmt.Errorf("unable to %s garage door, possible reason: mqttGdo availability reporting offline", action)
		} else if m.Settings.Topics.Obstruction != "" && m.Obstruction == "obstructed" {
			err = fmt.Errorf("unable to %s garage door, possible reason: mqttGdo obstruction reported", action)
		} else {
			err = fmt.Errorf("unable to %s garage door, possible reason: unknown; current state: %s", action, m.State)
		}
	} else {
		logger.Infof("Garage door command `%s` has been published to the topic", action)
	}

	return
}

func (m *mqttGdo) ProcessShutdown() {
	m.MqttClient.Disconnect(250)
}
