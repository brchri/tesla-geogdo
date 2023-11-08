package geo

import (
	"math"
)

type (
	// defines a center point and two radii (distances) to define open and close geofences
	CircularGeofence struct {
		Center        Point   `yaml:"center"`
		CloseDistance float64 `yaml:"close_distance"` // defines a radius from the center point; when vehicle moves from < distance to > distance, garage will close
		OpenDistance  float64 `yaml:"open_distance"`  // defines a radius from the center point; when vehicle moves from > distance to < distance, garage will open
	}
)

var circularMqttTopics = []string{"latitude", "longitude"}

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
	car.CurDistance = distance(car.CurrentLocation, car.GarageDoor.CircularGeofence.Center)

	// check if car has crossed a geofence and set an appropriate action
	if car.GarageDoor.CircularGeofence.CloseDistance > 0 && // is valid close distance defined
		prevDistance <= car.GarageDoor.CircularGeofence.CloseDistance &&
		car.CurDistance > car.GarageDoor.CircularGeofence.CloseDistance { // car was within close geofence, but now beyond it (car left geofence)
		action = ActionClose
	} else if car.GarageDoor.CircularGeofence.OpenDistance > 0 && // is valid open distance defined
		prevDistance >= car.GarageDoor.CircularGeofence.OpenDistance &&
		car.CurDistance < car.GarageDoor.CircularGeofence.OpenDistance { // car was outside of open geofence, but is now within it (car entered geofence)
		action = ActionOpen
	}
	return
}
