package geo

import (
	"errors"
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
		CircularGeofence  *CircularGeofence      `yaml:"circular_geofence"`
		TeslamateGeofence *TeslamateGeofence     `yaml:"teslamate_geofence"`
		PolygonGeofence   *PolygonGeofence       `yaml:"polygon_geofence"`
		Cars              []*Car                 `yaml:"cars"`   // cars housed within this garage
		OpenerConfig      map[string]interface{} `yaml:"opener"` // holds gdo config that is parsed on gdo.Initialize
		OpLock            bool                   // controls if garagedoor has been operated recently to prevent flapping
		GeofenceType      string                 // indicates whether garage door uses teslamate's geofence or not (checked during runtime)
		Geofence          GeofenceInterface
		Opener            gdo.GDO `yaml:"-"` // garage door opener; don't parse this from the garage door yaml
	}

	GeofenceInterface interface {
		getEventChangeAction(*Car) string
		GetMqttTopics() []string
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

	if action == "" || car.GarageDoor.OpLock {
		return // only execute if there's a valid action to execute and the garage door isn't on cooldown
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

		time.Sleep(time.Duration(util.Config.Global.OpCooldown) * time.Minute) // keep opLock true for OpCooldown minutes to prevent flapping in case of overlapping geofences
		car.GarageDoor.OpLock = false                                          // release garage door's operation lock
	}()
}

// checks for valid geofence values for a garage door
// preferred priority is polygon > circular > teslamate
// at least one open OR one close must be defined to identify a geofence type
func (g *GarageDoor) SetGeofenceType() error {
	var geoType string
	if g.PolygonGeofence != nil &&
		(len(g.PolygonGeofence.Open) > 0 ||
			len(g.PolygonGeofence.Close) > 0) {
		g.Geofence = g.PolygonGeofence
		geoType = "Polygon"
	} else if g.CircularGeofence != nil &&
		g.CircularGeofence.Center.IsPointDefined() &&
		(g.CircularGeofence.OpenDistance > 0 ||
			g.CircularGeofence.CloseDistance > 0) {
		g.Geofence = g.CircularGeofence
		geoType = "Circular"
	} else if g.TeslamateGeofence != nil &&
		(g.TeslamateGeofence.Close.IsTriggerDefined() ||
			g.TeslamateGeofence.Open.IsTriggerDefined()) {
		g.Geofence = g.TeslamateGeofence
		geoType = "Teslamate"
	} else {
		return errors.New("unable to determine geofence type for garage door")
	}
	logger.Debugf("Garage door geofence type identified: %s", geoType)
	return nil
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
		// check if kml_file was defined, and if so, load and parse kml and set polygon geofences accordingly
		if g.PolygonGeofence != nil && g.PolygonGeofence.KMLFile != "" {
			logger.Debugf("KML file %s found, loading", g.PolygonGeofence.KMLFile)
			if err := loadKMLFile(g.PolygonGeofence); err != nil {
				logger.Warnf("Unable to load KML file: %v", err)
			} else {
				logger.Debug("KML file loaded successfully")
			}
		}
		err = g.SetGeofenceType()
		if err != nil {
			logger.Fatalf("error: no supported geofences defined for garage door %v", g)
		} else {
			var geoType string
			switch g.Geofence.(type) {
			case *TeslamateGeofence:
				geoType = "Teslamate"
			case *PolygonGeofence:
				geoType = "Polygon"
			case *CircularGeofence:
				geoType = "Circular"
			}
			logger.Debugf("Garage door geofence type identified: %s", geoType)
		}

		g.Opener, err = InitializeGdoFunc(g.OpenerConfig)
		if err != nil {
			logger.Fatalf("Couldn't initialize garage door opener module, received error %s", err)
		}

		// initialize location update channel
		for _, c := range g.Cars {
			c.LocationUpdate = make(chan Point, 2)
		}
	}
}
