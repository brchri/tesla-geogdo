package geo

import (
	"encoding/xml"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	util "github.com/brchri/tesla-geogdo/internal/util"
	logger "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type (
	// contains 3 geofences, open, close, and restricted, each of which are a list of lat/long points defining the polygon
	PolygonGeofence struct {
		Close      []Point `yaml:"close,omitempty"`      // list of points defining a polygon; when vehicle moves from inside this geofence to outside, garage will close
		Open       []Point `yaml:"open,omitempty"`       // list of points defining a polygon; when vehicle moves from outside this geofence to inside, garage will open
		Restricted []Point `yaml:"restricted,omitempty"` // list of points defining a polygon; when vehicle moves from inside this geofence to inside open geofence, garage will not open
		KMLFile    string  `yaml:"kml_file,omitempty"`
	}

	// kml schema to parse coordinates from kml file for polygon geofences
	KML struct {
		Document struct {
			Placemarks []struct {
				Name    string `xml:"name"`
				Polygon struct {
					OuterBoundary struct {
						LinearRing struct {
							Coordinates string `xml:"coordinates"`
						} `xml:"linearring"`
					} `xml:"outerboundaryis"`
				} `xml:"polygon"`
			} `xml:"placemark"`
		} `xml:"document"`
	}
)

func init() {
	logger.SetFormatter(&util.CustomFormatter{})
	logger.SetOutput(os.Stdout)
	if val, ok := os.LookupEnv("DEBUG"); ok && strings.ToLower(val) == "true" {
		logger.SetLevel(logger.DebugLevel)
	}
}

// get action based on whether we had a polygon geofence change event
// uses ray-casting algorithm, assumes a simple geofence (no holes or border cross points)
func (p *PolygonGeofence) getEventChangeAction(tracker *Tracker) (action string) {
	if !tracker.CurrentLocation.IsPointDefined() {
		return // need valid lat and long to check geofence
	}

	isInsideCloseGeo := isInsidePolygonGeo(tracker.CurrentLocation, p.Close)
	isInsideOpenGeo := isInsidePolygonGeo(tracker.CurrentLocation, p.Open)
	isInsideRestrictedGeo := false
	if len(p.Restricted) > 0 {
		isInsideRestrictedGeo = isInsidePolygonGeo(tracker.CurrentLocation, p.Restricted)
	}

	if len(p.Close) > 0 {
		if tracker.InsidePolyCloseGeo && !tracker.InsidePolyRestrictedGeo && !isInsideCloseGeo { // if we were inside the close geofence and now we're not, then close (if also not coming from a restricted zone)
			action = ActionClose
		} else if !tracker.InsidePolyCloseGeo && isInsideCloseGeo { // if we just entered the close geo, then set LastNoOpEvent to prevent flapping and accidentally triggering an open
			tracker.LastEnteredCloseGeo = time.Now()
		}
	}
	if len(p.Open) > 0 {
		if !tracker.InsidePolyOpenGeo && !tracker.InsidePolyRestrictedGeo && isInsideOpenGeo { // if we were not inside the open geo or the restricted geo, and now we are in the open geo, then open
			action = ActionOpen
		} else if tracker.InsidePolyOpenGeo && !isInsideOpenGeo { // if we just left the open geo, then set LastNoOpEvent to prevent flapping and accidentally triggering an open
			tracker.LastLeftOpenGeo = time.Now()
		}
	}

	tracker.InsidePolyCloseGeo = isInsideCloseGeo
	tracker.InsidePolyOpenGeo = isInsideOpenGeo
	tracker.InsidePolyRestrictedGeo = isInsideRestrictedGeo

	return
}

func isInsidePolygonGeo(p Point, geofence []Point) bool {
	var intersections int
	j := len(geofence) - 1

	for i := 0; i < len(geofence); i++ {
		if ((geofence[i].Lat > p.Lat) != (geofence[j].Lat > p.Lat)) &&
			p.Lng < (geofence[j].Lng-geofence[i].Lng)*(p.Lat-geofence[i].Lat)/(geofence[j].Lat-geofence[i].Lat)+geofence[i].Lng {
			intersections++
		}
		j = i
	}

	return intersections%2 == 1 // are we currently inside a polygon geo
}

// loads kml file and overrides polygon geofence points with parsed data
func loadKMLFile(p *PolygonGeofence) error {
	fileContent, err := os.ReadFile(p.KMLFile)
	lowerKML := strings.ToLower(string(fileContent)) // convert xml to lower to make xml tag parsing case insensitive
	if err != nil {
		logger.Infof("Could not read file %s, received error: %v", p.KMLFile, err)
		return err
	}

	var kml KML
	err = xml.Unmarshal([]byte(lowerKML), &kml)
	if err != nil {
		logger.Infof("Could not load kml from file %s, received error: %v", p.KMLFile, err)
		return err
	}

	// loop through placemarks to get name and, if relevant, parse the coordinates accordingly
	for _, placemark := range kml.Document.Placemarks {
		var polygonGeoPoints []Point
		// geofences must be named `open` or `close` or they're considered irrelevant
		if placemark.Name != "open" && placemark.Name != "close" && placemark.Name != "restricted" {
			continue
		}

		for _, c := range strings.Split(placemark.Polygon.OuterBoundary.LinearRing.Coordinates, "\n") {
			// trim whitespace and continue loop if nothing is left
			c = strings.TrimSpace(c)
			if c == "" {
				continue
			}

			// kml coordinate format is longitude,latitude; split comma delim and parse coords
			coords := strings.Split(c, ",")
			lat, err := strconv.ParseFloat(coords[1], 64)
			if err != nil {
				logger.Infof("Could not parse lng/lat coordinates from line %s, received error: %v", c, err)
				return err
			}
			lng, err := strconv.ParseFloat(coords[0], 64)
			if err != nil {
				logger.Infof("Could not parse lng/lat coordinates from line %s, received error: %v", c, err)
				return err
			}

			polygonGeoPoints = append(polygonGeoPoints, Point{Lat: lat, Lng: lng})
		}

		// set either open or close polygon geo for garage door based on Placemark's Name element
		switch placemark.Name {
		case "open":
			p.Open = polygonGeoPoints
		case "close":
			p.Close = polygonGeoPoints
		case "restricted":
			p.Restricted = polygonGeoPoints
		}
	}

	return nil
}

func (p *PolygonGeofence) parseSettings(config map[string]interface{}) error {
	yamlData, err := yaml.Marshal(config)
	var settings PolygonGeofence
	if err != nil {
		return fmt.Errorf("failed to marshal geofence yaml object, error: %v", err)
	}
	err = yaml.Unmarshal(yamlData, &settings)
	if err != nil {
		return fmt.Errorf("failed to unmarshal geofence yaml object, error: %v", err)
	}
	*p = settings
	if p.KMLFile != "" {
		return loadKMLFile(p)
	}
	return nil
}
