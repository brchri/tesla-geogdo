package geo

import (
	"fmt"
	"math"

	"gopkg.in/yaml.v3"
)

type (
	// defines a center point and two radii (distances) to define open and close geofences
	CircularGeofence struct {
		Center        Point   `yaml:"center"`
		CloseDistance float64 `yaml:"close_distance"` // defines a radius from the center point; when vehicle moves from < distance to > distance, garage will close
		OpenDistance  float64 `yaml:"open_distance"`  // defines a radius from the center point; when vehicle moves from > distance to < distance, garage will open
	}
)

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
	if c.CloseDistance > 0 && // is valid close distance defined
		prevDistance <= c.CloseDistance &&
		tracker.CurDistance > c.CloseDistance { // tracker was within close geofence, but now beyond it (tracker left geofence)
		action = ActionClose
	} else if c.OpenDistance > 0 && // is valid open distance defined
		prevDistance >= c.OpenDistance &&
		tracker.CurDistance < c.OpenDistance { // tracker was outside of open geofence, but is now within it (tracker entered geofence)
		action = ActionOpen
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
