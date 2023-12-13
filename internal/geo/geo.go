package geo

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/brchri/tesla-geogdo/internal/gdo"
	util "github.com/brchri/tesla-geogdo/internal/util"
	logger "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type (
	Point struct {
		Lat float64 `yaml:"lat"`
		Lng float64 `yaml:"lng"`
	}

	Car struct {
		ID                 int         `yaml:"teslamate_car_id"` // mqtt identifier for vehicle
		GarageDoor         *GarageDoor // bidirectional pointer to GarageDoor containing car
		CurrentLocation    Point       // current vehicle location
		LocationUpdate     chan Point  // channel to receive location updates
		CurDistance        float64     // current distance from garagedoor location
		PrevGeofence       string      // geofence previously ascribed to car
		CurGeofence        string      // updated geofence ascribed to car when published to mqtt
		InsidePolyOpenGeo  bool        // indicates if car is currently inside the polygon_open_geofence
		InsidePolyCloseGeo bool        // indicates if car is currently inside the polygon_close_geofence
	}

	// defines a garage door with one unique geofence type: circular, teslamate, or polygon
	// only one geofence type may be defined per garage door
	// if more than one defined, priority will be polygon > circular > teslamate
	GarageDoor struct {
		Geofence       GeofenceInterface      `yaml:"-"` // geofence; don't parse this from the geofence yaml
		Opener         gdo.GDO                `yaml:"-"` // garage door opener; don't parse this from the garage door yaml
		GeofenceConfig map[string]interface{} `yaml:"geofence"`
		OpenerConfig   map[string]interface{} `yaml:"opener"` // holds gdo config that is parsed on gdo.Initialize
		Cars           []*Car                 `yaml:"cars"`   // cars housed within this garage
		OpLock         bool                   // controls if garagedoor has been operated recently to prevent flapping
	}

	// interface to represent geofence object
	GeofenceInterface interface {
		// check for an event trigger if a geofence is crossed and return appropriate action
		// determines if a car is currently within a geofence, if it was previously,
		// and what action should be taken if those are different (indicating a crossing of geofences)
		getEventChangeAction(*Car) string
		// get teslamate mqtt topics relevant to the implemented geofence type
		GetMqttTopics(carId int) []string
		// parse the settings: of a geofence into the specific geofence type struct
		parseSettings(map[string]interface{}) error
	}
)

const (
	ActionOpen  = "open"
	ActionClose = "close"
)

var (
	GarageDoors       []*GarageDoor
	InitializeGdoFunc = gdo.Initialize // abstract gdo.Initialize function call to allow mocking
)

func init() {
	logger.SetFormatter(&util.CustomFormatter{})
	logger.SetOutput(os.Stdout)
	if val, ok := os.LookupEnv("DEBUG"); ok && strings.ToLower(val) == "true" {
		logger.SetLevel(logger.DebugLevel)
	}
}

func (p Point) IsPointDefined() bool {
	// lat=0 lng=0 are valid coordinates, but they're in the middle of the ocean, so safe to assume these mean undefined
	return p.Lat != 0 && p.Lng != 0
}

// check if outside close geo or inside open geo and set garage door state accordingly
func CheckGeofence(car *Car) {

	// get action based on either geo cross events or distance threshold cross events
	action := car.GarageDoor.Geofence.getEventChangeAction(car)

	if action == "" {
		return // nothing to do
	}
	if car.GarageDoor.OpLock {
		logger.Debugf("Garage operation is locked (due to either cooldown or current activity), will not execute action '%s'", action)
		return
	}

	car.GarageDoor.OpLock = true // set lock so no other threads try to operate the garage before the cooldown period is complete
	// send operation to garage door and wait for timeout to release oplock
	// run as goroutine to prevent blocking update channels from mqtt broker in main
	go func() {
		switch car.GarageDoor.Geofence.(type) {
		case *TeslamateGeofence:
			logger.Infof("Attempting to %s garage door for car %d", action, car.ID)
		default:
			logger.Infof("Attempting to %s garage door for car %d at lat %f, long %f", action, car.ID, car.CurrentLocation.Lat, car.CurrentLocation.Lng)
		}

		// create retry loop to set the garage door state
		for i := 1; i > 0; i-- { // temporarily setting to 1 to disable retry logic while myq auth endpoint stabilizes to avoid rate limiting
			err := car.GarageDoor.Opener.SetGarageDoor(action)
			if err == nil {
				// no error received, so breaking retry loop)
				break
			}
			logger.Error(err)
			if i == 1 {
				logger.Warn("No further attempts will be made")
			} else {
				logger.Warnf("Retrying set garage door state %d more time(s)", i-1)
			}
		}

		if util.Config.Global.OpCooldown > 0 {
			time.Sleep(time.Duration(util.Config.Global.OpCooldown) * time.Minute) // keep opLock true for OpCooldown minutes to prevent flapping in case of overlapping geofences
		} else if os.Getenv("GDO_SKIP_FLAP_DELAY") != "true" {
			// because lat and long are processed individually, it's possible that a car may flap briefly on the geofence crossing which can spam action calls to the gdo
			// add a small sleep to prevent this
			logger.Debugf("Garage for car %d retaining oplock for 5s to mitigate flapping when crossing geofence...", car.ID)
			time.Sleep(5000 * time.Millisecond)
		}
		car.GarageDoor.OpLock = false // release garage door's operation lock
	}()
}

func ParseGarageDoorConfig() {
	// marshall map[string]interface into yaml, then unmarshal to object based on yaml def in struct
	yamlData, err := yaml.Marshal(util.Config.GarageDoors)
	if err != nil {
		logger.Fatal("Failed to marhsal garage doors yaml object")
	}
	err = yaml.Unmarshal(yamlData, &GarageDoors)
	if err != nil {
		logger.Fatal("Failed to unmarhsal garage doors yaml object")
	}

	logger.Debug("Checking garage door configs")
	if len(GarageDoors) == 0 {
		logger.Fatal("Unable to find garage doors in config! Please ensure proper spacing in the config file")
	}
	for i, g := range GarageDoors {
		if len(g.Cars) == 0 {
			logger.Fatalf("No cars found for garage door #%d! Please ensure proper spacing in the config file", i)
		}

		g.Geofence, err = newGeofence(g.GeofenceConfig)
		if err != nil {
			logger.Fatalf("unable to parse geofence config for door %d, received error: %v", i, err)
		}

		g.Opener, err = InitializeGdoFunc(g.OpenerConfig)
		if err != nil {
			logger.Fatalf("Couldn't initialize garage door opener module, received error %s", err)
		}

		// initialize location update channel
		for _, c := range g.Cars {
			c.LocationUpdate = make(chan Point)
		}
	}
}

// return a new instance of a GeofenceInterface based on the type defined in the config yml
func newGeofence(config map[string]interface{}) (GeofenceInterface, error) {
	type geofenceConfig struct {
		GeofenceType string                 `yaml:"type"`
		Settings     map[string]interface{} `yaml:"settings"`
	}
	var geoConfig geofenceConfig
	// marshall map[string]interface into yaml, then unmarshal to object based on yaml def in struct
	yamlData, err := yaml.Marshal(config)
	if err != nil {
		logger.Fatal("Failed to marhsal geofence yaml object")
	}
	err = yaml.Unmarshal(yamlData, &geoConfig)
	if err != nil {
		logger.Fatal("Failed to unmarhsal geofence yaml object")
	}
	var g GeofenceInterface
	switch geoConfig.GeofenceType {
	case "circular":
		g = &CircularGeofence{}
	case "teslamate":
		g = &TeslamateGeofence{}
	case "polygon":
		g = &PolygonGeofence{}
	default:
		return nil, fmt.Errorf("unable to parse geofence config type %s", geoConfig.GeofenceType)
	}
	return g, g.parseSettings(geoConfig.Settings)
}
