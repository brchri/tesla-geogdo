package geo

import (
	"fmt"
	"math"

	util "github.com/brchri/tesla-geogdo/internal/util"
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

var circularMqttTopics = []string{
	util.Config.Global.MqttSettings.LatTopic,
	util.Config.Global.MqttSettings.LngTopic,
}

func (c *CircularGeofence) GetMqttTopics() []string {
	return circularMqttTopics
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
func (c *CircularGeofence) getEventChangeAction(car *Car) (action string) {
	if !car.CurrentLocation.IsPointDefined() {
		return // need valid lat and lng to check fence
	}

	// update car's current distance, and store the previous distance in a variable
	prevDistance := car.CurDistance
	car.CurDistance = distance(car.CurrentLocation, c.Center)

	// check if car has crossed a geofence and set an appropriate action
	if c.CloseDistance > 0 && // is valid close distance defined
		prevDistance <= c.CloseDistance &&
		car.CurDistance > c.CloseDistance { // car was within close geofence, but now beyond it (car left geofence)
		action = ActionClose
	} else if c.OpenDistance > 0 && // is valid open distance defined
		prevDistance >= c.OpenDistance &&
		car.CurDistance < c.OpenDistance { // car was outside of open geofence, but is now within it (car entered geofence)
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
