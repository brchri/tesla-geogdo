package geo

import (
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	util "github.com/brchri/tesla-geogdo/internal/util"
	logger "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type (
	// defines a center point and two radii (distances) to define open and close geofences
	CircularGeofence struct {
		Center        Point   `yaml:"center"`
		CloseDistance float64 `yaml:"close_distance,omitempty"` // defines a radius from the center point; when vehicle moves from < distance to > distance, garage will close
		OpenDistance  float64 `yaml:"open_distance,omitempty"`  // defines a radius from the center point; when vehicle moves from > distance to < distance, garage will open
	}
)

func init() {
	logger.SetFormatter(&util.CustomFormatter{})
	logger.SetOutput(os.Stdout)
	if val, ok := os.LookupEnv("DEBUG"); ok && strings.ToLower(val) == "true" {
		logger.SetLevel(logger.DebugLevel)
	}
}

func distance(point1 Point, point2 Point) float64 {
	// Calculate the distance between two points using the haversine formula
	const radius = 6371 // Earth's radius in kilometers
	lat1 := toRadians(point1.Lat)
	lat2 := toRadians(point2.Lat)
	deltaLat := toRadians(point2.Lat - point1.Lat)
	deltaLon := toRadians(point2.Lng - point1.Lng)
	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) + math.Cos(lat1)*math.Cos(lat2)*math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	d := radius * c
	return d
}

func toRadians(degrees float64) float64 {
	return degrees * math.Pi / 180
}

// gets action based on if there was a relevant distance change
func (c *CircularGeofence) getEventChangeAction(tracker *Tracker) (action string) {
	if !tracker.CurrentLocation.IsPointDefined() {
		return // need valid lat and lng to check fence
	}

	// update tracker's current distance, and store the previous distance in a variable
	prevDistance := tracker.CurDistance
	tracker.CurDistance = distance(tracker.CurrentLocation, c.Center)

	// check if tracker has crossed a geofence and set an appropriate action
	if c.CloseDistance > 0 { // is valid close distance defined
		if prevDistance <= c.CloseDistance &&
			tracker.CurDistance > c.CloseDistance { // tracker was within close geofence, but now beyond it (tracker left geofence)
			logger.Debugf("Tracker left close geofence: Close Radius: %v; PreviousDistance: %v; CurrentDistance: %v", c.CloseDistance, prevDistance, tracker.CurDistance)
			action = ActionClose
		}
		if prevDistance > c.CloseDistance &&
			tracker.CurDistance <= c.CloseDistance { // tracker just entered close geofence
			logger.Debugf("Tracker entered close geofence: Close Radius: %v; PreviousDistance: %v; CurrentDistance: %v", c.CloseDistance, prevDistance, tracker.CurDistance)
			tracker.LastEnteredCloseGeo = time.Now()
		}
	}
	if c.OpenDistance > 0 { // is valid open distance defined
		if prevDistance >= c.OpenDistance &&
			tracker.CurDistance < c.OpenDistance { // tracker was outside of open geofence, but is now within it (tracker entered geofence)
			logger.Debugf("Tracker entered open geofence: Open Radius: %v; PreviousDistance: %v; CurrentDistance: %v", c.OpenDistance, prevDistance, tracker.CurDistance)
			action = ActionOpen
		} else if prevDistance < c.OpenDistance &&
			tracker.CurDistance >= c.OpenDistance { // tracker just left open geofence
			logger.Debugf("Tracker left open geofence: Open Radius: %v; PreviousDistance: %v; CurrentDistance: %v", c.OpenDistance, prevDistance, tracker.CurDistance)
			tracker.LastLeftOpenGeo = time.Now()
		}
	}
	return
}

func (c *CircularGeofence) parseSettings(config map[string]interface{}) error {
	yamlData, err := yaml.Marshal(config)
	var settings CircularGeofence
	if err != nil {
		return fmt.Errorf("failed to marhsal geofence yaml object, error: %v", err)
	}
	err = yaml.Unmarshal(yamlData, &settings)
	if err != nil {
		return fmt.Errorf("failed to unmarhsal geofence yaml object, error: %v", err)
	}
	*c = settings
	return nil
}
