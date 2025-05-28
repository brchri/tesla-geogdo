package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/brchri/tesla-geogdo/cmd/app/console"
	"github.com/brchri/tesla-geogdo/internal/geo"
	"github.com/google/uuid"
	logger "github.com/sirupsen/logrus"

	"github.com/brchri/tesla-geogdo/internal/util"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var (
	configFile   string
	trackers     []*geo.Tracker            // list of all trackers from all garage doors
	version      string         = "v0.0.1" // pass -ldflags="-X main.version=<version>" at build time to set linker flag and bake in binary version
	commitHash   string
	messageChan  chan mqtt.Message         // channel to receive mqtt messages
	mqttSettings *util.MqttConnectSettings // point to util.Config.Global.MqttSettings.Connection for shorter reference
	pauseChan    chan int                  // handles sending message to goroutine that pauses operations based on api calls
)

func init() {
	logger.SetFormatter(&util.CustomFormatter{})
	logger.SetOutput(os.Stdout)
	if val, ok := os.LookupEnv("DEBUG"); ok && strings.ToLower(val) == "true" {
		logger.SetLevel(logger.DebugLevel)
	}
	log.SetOutput(os.Stdout)

	parseArgs()
	util.LoadConfig(configFile)
	mqttSettings = &util.Config.Global.MqttSettings.Connection
	if util.Config.Testing {
		logger.Warn("TESTING=true, will not execute garage door actions")
	}

	geo.ParseGarageDoorConfig()
	checkEnvVars()
	for _, garageDoor := range geo.GarageDoors {
		for _, tracker := range garageDoor.Trackers {
			tracker.GarageDoor = garageDoor
			trackers = append(trackers, tracker)
			tracker.InsidePolyCloseGeo = true // only relevent for polygon geos but won't be used if that's not the geofence type
			tracker.InsidePolyOpenGeo = true  // only relevent for polygon geos but won't be used if that's not the geofence type
			// start listening to tracker update location channels
			go processLocationUpdates(tracker)
		}
	}
}

// parse args
func parseArgs() {
	// set up flags for parsing args
	var getVersion bool
	var runWizardFlag bool
	flag.StringVar(&configFile, "config", "", "location of config file")
	flag.StringVar(&configFile, "c", "", "location of config file")
	flag.BoolVar(&util.Config.Testing, "testing", false, "test case")
	flag.BoolVar(&getVersion, "v", false, "print version info and return")
	flag.BoolVar(&getVersion, "version", false, "print version info and return")
	flag.BoolVar(&runWizardFlag, "wizard", false, "run the config wizard")
	flag.Parse()

	if getVersion {
		versionInfo := fmt.Sprintf("%s %s %s/%s; commit hash %s", filepath.Base(os.Args[0]), version, runtime.GOOS, runtime.GOARCH, commitHash)
		fmt.Println(versionInfo)
		os.Exit(0)
	}

	if runWizardFlag {
		console.RunWizard()
		os.Exit(0)
	}

	// if -c or --config wasn't passed, check for CONFIG_FILE env var
	// if that fails, check for file at default location
	if configFile == "" {
		var exists bool
		if configFile, exists = os.LookupEnv("CONFIG_FILE"); !exists {
			logger.Fatalf("Config file must be defined with '-c' or 'CONFIG_FILE' environment variable")
		}
	}

	// check that ConfigFile exists
	if _, err := os.Stat(configFile); err != nil {
		logger.Fatalf("Config file %v doesn't exist!", configFile)
	}
}

func main() {

	// initialize api handlers
	pauseChan = make(chan int)
	http.HandleFunc("/pause", apiPauseHandler)
	http.HandleFunc("/resume", apiPauseHandler)
	go http.ListenAndServe(":8555", nil)

	messageChan = make(chan mqtt.Message)

	logger.Debug("Setting MQTT Opts:")
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
	if mqttSettings.User != "" {
		logger.Debug(" Username: true <redacted value>")
	} else {
		logger.Debug(" Username: false (not set)")
	}
	opts.SetUsername(mqttSettings.User) // if not defined, will just set empty strings and won't be used by pkg
	if mqttSettings.Pass != "" {
		logger.Debug(" Password: true <redacted value>")
	} else {
		logger.Debug(" Password: false (not set)")
	}
	opts.SetPassword(mqttSettings.Pass) // if not defined, will just set empty strings and won't be used by pkg
	opts.OnConnect = onMqttConnect

	// set conditional MQTT client opts
	if mqttSettings.ClientID != "" {
		logger.Debugf(" ClientID: %s", mqttSettings.ClientID)
		opts.SetClientID(mqttSettings.ClientID)
	} else {
		// generate UUID for mqtt client connection if not specified in config file
		id := uuid.New().String()
		logger.Debugf(" ClientID: %s", id)
		opts.SetClientID(id)
	}
	logger.Debug(" Protocol: TCP")
	mqttProtocol := "tcp"
	if mqttSettings.UseTls {
		logger.Debug(" UseTLS: true")
		logger.Debugf(" SkipTLSVerify: %t", mqttSettings.SkipTlsVerify)
		opts.SetTLSConfig(&tls.Config{
			InsecureSkipVerify: mqttSettings.SkipTlsVerify,
		})
		mqttProtocol = "ssl"
	} else {
		logger.Debug(" UseTLS: false")
	}
	broker := fmt.Sprintf("%s://%s:%d", mqttProtocol, mqttSettings.Host, mqttSettings.Port)
	logger.Debugf(" Broker: %s", broker)
	opts.AddBroker(broker)

	// create a new MQTT client object
	client := mqtt.NewClient(opts)

	// connect to the MQTT broker
	logger.Debug("Connecting to MQTT broker")
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		logger.Fatalf("could not connect to mqtt broker: %v", token.Error())
	} else {
		logger.Info("Connected to Teslamate MQTT broker")
	}
	logger.Debugf("MQTT Broker Connected: %t", client.IsConnected())

	// listen for incoming messages
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)

	for {
		select {
		case message := <-messageChan:

		topic:
			// check if topic matches any trackers and execute action
			for _, t := range trackers {
				var point geo.Point
				var err error
				switch message.Topic() {
				case t.LatTopic:
					logger.Debugf("Received lat for tracker %v: %s", t.ID, string(message.Payload()))
					point.Lat, err = strconv.ParseFloat(string(message.Payload()), 64)
				case t.LngTopic:
					logger.Debugf("Received long for tracker %v: %s", t.ID, string(message.Payload()))
					point.Lng, err = strconv.ParseFloat(string(message.Payload()), 64)
				case t.GeofenceTopic:
					t.PrevGeofence = t.CurGeofence
					t.CurGeofence = string(message.Payload())
					logger.Infof("Received geo for tracker %v: %s", t.ID, t.CurGeofence)
					go geo.CheckGeofence(t)
				case t.ComplexTopic.Topic:
					logger.Debugf("Received payload for complex topic %s for tracker %v, payload:\n%s", message.Topic(), t.ID, string(message.Payload()))
					point, err = processComplexTopicPayload(t, string(message.Payload()))
				default:
					continue topic // no topic match for this tracker found, move on to next tracker
				}

				if err != nil {
					logger.Errorf("could not parse message payload from topic for tracker %v, received error %v", t.ID, err)
				}

				// if a point is now defined, process a location update and stop looking for matching topics
				if point != (geo.Point{}) {
					go func(p geo.Point, t *geo.Tracker) {
						// send as goroutine so it doesn't block other vehicle updates if channel buffer is full
						t.LocationUpdate <- p
					}(point, t)
				}
			}

		case <-signalChannel:
			logger.Info("Received interrupt signal, shutting down...")
			client.Disconnect(250)
			for _, g := range geo.GarageDoors {
				g.Opener.ProcessShutdown()
			}
			time.Sleep(250 * time.Millisecond)
			return

		}
	}
}

func processComplexTopicPayload(tracker *geo.Tracker, payload string) (geo.Point, error) {
	var jsonData map[string]interface{}
	var p geo.Point
	err := json.Unmarshal([]byte(payload), &jsonData)
	if err != nil {
		return geo.Point{}, fmt.Errorf("could not unmarshal json string to map object")
	}
	payloadType, ok := jsonData["_type"].(string)
	if ok && payloadType == "lwt" {
		logger.Debugf("Payload for tracker %v is '_type: lwt'; will not process location update", tracker.ID)
		return p, nil
	}
	lat, ok := jsonData[tracker.ComplexTopic.LatJsonKey].(float64)
	if ok {
		p.Lat = lat
	}
	lng, ok := jsonData[tracker.ComplexTopic.LngJsonKey].(float64)
	if ok {
		p.Lng = lng
	}

	if p.Lat == 0 && p.Lng == 0 {
		return p, fmt.Errorf("could not parse lat or lon from complex topic message")
	}
	return p, nil
}

// watches the LocationUpdate channel for a tracker and queues a CheckGeofence operation
// this allows threaded geofence checks for multiple vehicles, while each individual vehicle
// does not have parallel threads executing checks
func processLocationUpdates(tracker *geo.Tracker) {
	for update := range tracker.LocationUpdate {
		var newLocation bool
		if update.Lat != 0 {
			tracker.CurrentLocation.Lat = update.Lat
			newLocation = true
		}
		if update.Lng != 0 {
			tracker.CurrentLocation.Lng = update.Lng
			newLocation = true
		}
		if newLocation && tracker.CurrentLocation.IsPointDefined() {
			geo.CheckGeofence(tracker)
		}
	}
}

// subscribe to topics when MQTT client connects (or reconnects)
func onMqttConnect(client mqtt.Client) {
	for _, tracker := range trackers {
		logger.Infof("Subscribing to MQTT topics for tracker %v", tracker.ID)

		// define which topics are relevant for each tracker based on config
		topics := []string{}
		for _, t := range []string{
			tracker.LatTopic,
			tracker.LngTopic,
			tracker.GeofenceTopic,
			tracker.ComplexTopic.Topic,
		} {
			if t != "" {
				topics = append(topics, t)
			}
		}

		// subscribe to topics
		for _, topic := range topics {
			topicSubscribed := false
			// retry topic subscription attempts with 1 sec delay between attempts
			for retryAttempts := 5; retryAttempts > 0; retryAttempts-- {
				logger.Debugf("Subscribing to topic: %s", topic)
				if token := client.Subscribe(
					topic,
					0,
					func(client mqtt.Client, message mqtt.Message) {
						messageChan <- message
					}); token.Wait() && token.Error() == nil {
					topicSubscribed = true
					logger.Debugf("Topic subscribed successfully: %s", topic)
					break
				} else {
					logger.Infof("Failed to subscribe to topic %s for tracker %v, will make %d more attempts. Error: %v", topic, tracker.ID, retryAttempts, token.Error())
				}
				time.Sleep(5 * time.Second)
			}
			if !topicSubscribed {
				logger.Fatalf("Unable to subscribe to topics, exiting")
			}
		}
	}

	logger.Info("Topics subscribed, listening for events...")
}

// check for env vars and validate that a myq_email and myq_pass exists
func checkEnvVars() {
	logger.Debug("Checking environment variables:")
	// override config with env vars if present
	if value, exists := os.LookupEnv("TRACKER_MQTT_USER"); exists {
		logger.Debug("  TRACKER_MQTT_USER defined, overriding config")
		mqttSettings.User = value
	}
	if value, exists := os.LookupEnv("TRACKER_MQTT_PASS"); exists {
		logger.Debug("  TRACKER_MQTT_PASS defined, overriding config")
		mqttSettings.Pass = value
	}
	if value, exists := os.LookupEnv("TESTING"); exists {
		util.Config.Testing, _ = strconv.ParseBool(value)
		logger.Debugf("  TESTING=%t", util.Config.Testing)
	}
	if value, exists := os.LookupEnv("DEBUG"); exists {
		logger.Debugf("  DEBUG=%s", value)
	}
}

// receives api requests related to pause and resume functions
// expects GET requests at either the /pause or /resume endpoints
// and sends to relevant helper functions for processing
func apiPauseHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	if r.URL.Path == "/resume" {
		resumeOperations()
		return
	}

	query := r.URL.Query()
	duration := query.Get("duration")
	var durationInt int
	if duration != "" && duration != "0" {
		var err error
		durationInt, err = strconv.Atoi(duration)
		if err != nil {
			http.Error(w, "Invalid duration parameter", http.StatusBadRequest)
			return
		}
	}
	pauseOperations(durationInt)
}

// pauses garage operations either indefinitely or for a finite duration
// all other processing still functions (e.g. tracking, geofence awareness, etc),
// only garage operations are disabled
//
// if a finite duration is provided, a goroutine will be initiated to wait for the duration
// to timeout and re-enable garage operations. this goroutine also monitors a go channel
// in case the pause duration is to be overriden, either by a new duration (finite or infinite),
// or by a resume command, and behaves accordingly
func pauseOperations(duration int) {
	if duration == 0 {
		duration = -1 // if no duration was defined, set to -1 for infinite pause
	}
	if duration > 0 {
		logger.Infof("Received request to pause operations, pausing for %d seconds, use /resume endpoint to resume garage operations sooner than indicated time", duration)
	} else {
		logger.Info("Received request to pause operations indefinitely; use /resume endpoint to resume garage operations")
	}

	if util.Config.MasterOpLock > 0 { // if we have a finite lock in progress, send new duration to channel
		pauseChan <- duration
		return
	}
	util.Config.MasterOpLock = duration

	// only set a timeout loop if duration > 0, negatives are infinite pauses
	if util.Config.MasterOpLock > 0 {
		go func() {
			for ; util.Config.MasterOpLock > 0; util.Config.MasterOpLock-- {
				time.Sleep(1 * time.Second)

				// non-blocking select to check for channel message indicating a resume api call has been made and we can break the loop
				select {
				case msg := <-pauseChan:
					util.Config.MasterOpLock = msg
					if msg <= 0 {
						// either received an indefinite pause (<0) or a resume (=0), so loop with unlock final action is no longer needed
						return
					}
				default:
				}
			}
			logger.Info("Pause timeout reached; unpausing operations")
			util.Config.MasterOpLock = 0
		}()
	}
}

// helper function to resume garage operations
// if there is currently a finite pause in progress, it will send
// the new finite duration to the channel to be consumed by the currently
// runnin goroutine with the updated value; else it will set the lock back to 0,
// which is the disabled value (thereby resuming garage operations)
func resumeOperations() {
	logger.Info("Received request to resume operations, resuming...")
	if util.Config.MasterOpLock > 0 {
		// send signal to pause timeout loop it's no longer needed
		// send as goroutine as we only read channel every 1 second, so this ensures fast api response while waiting for channel to be read by loop
		go func() { pauseChan <- 0 }()
	} else if util.Config.MasterOpLock < 0 {
		util.Config.MasterOpLock = 0 // override indefinite pause
	}
}
