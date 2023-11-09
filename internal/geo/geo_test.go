package geo

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/brchri/tesla-geogdo/internal/gdo"
	"github.com/brchri/tesla-geogdo/internal/mocks"
	"github.com/stretchr/testify/assert"

	"github.com/brchri/tesla-geogdo/internal/util"
)

var (
	distanceCar        *Car
	distanceGarageDoor *GarageDoor
	distanceGeofence   *CircularGeofence

	teslamateGarageDoor *GarageDoor
	teslamateCar        *Car
	teslamateGeofence   *TeslamateGeofence

	polygonGarageDoor *GarageDoor
	polygonCar        *Car
	polygonGeofence   *PolygonGeofence
)

func init() {
	util.LoadConfig(filepath.Join("..", "..", "examples", "config.multiple.yml"))
	InitializeGdoFunc = func(config map[string]interface{}) (gdo.GDO, error) { return &mocks.GDO{}, nil }
	ParseGarageDoorConfig()

	// used for testing events based on distance
	distanceGarageDoor = GarageDoors[0]
	distanceCar = distanceGarageDoor.Cars[0]
	distanceCar.GarageDoor = distanceGarageDoor
	distanceGeofence, _ = distanceCar.GarageDoor.Geofence.(*CircularGeofence) // type cast geofence interface

	// used for testing events based on teslamate geofence changes
	teslamateGarageDoor = GarageDoors[1]
	teslamateCar = teslamateGarageDoor.Cars[0]
	teslamateCar.GarageDoor = teslamateGarageDoor
	teslamateGeofence, _ = teslamateCar.GarageDoor.Geofence.(*TeslamateGeofence) // type cast geofence interface

	// used for testing events based on teslamate geofence changes
	polygonGarageDoor = GarageDoors[2]
	polygonCar = polygonGarageDoor.Cars[0]
	polygonCar.GarageDoor = polygonGarageDoor
	polygonGeofence, _ = polygonCar.GarageDoor.Geofence.(*PolygonGeofence) // type cast geofence interface

	util.Config.Global.OpCooldown = 0
}

func Test_getEventChangeAction_Circular(t *testing.T) {
	distanceCar.CurDistance = 0
	distanceCar.CurrentLocation.Lat = distanceGeofence.Center.Lat + 10
	distanceCar.CurrentLocation.Lng = distanceGeofence.Center.Lng

	assert.Equal(t, ActionClose, distanceCar.GarageDoor.Geofence.getEventChangeAction(distanceCar))
	assert.Greater(t, distanceCar.CurDistance, distanceGeofence.CloseDistance)

	distanceCar.CurrentLocation.Lat = distanceGeofence.Center.Lat

	assert.Equal(t, ActionOpen, distanceCar.GarageDoor.Geofence.getEventChangeAction(distanceCar))
	assert.Less(t, distanceCar.CurDistance, distanceGeofence.OpenDistance)
}

func Test_getEventChangeAction_Teslamate(t *testing.T) {
	teslamateCar.PrevGeofence = "home"
	teslamateCar.CurGeofence = "not_home"

	assert.Equal(t, ActionClose, teslamateCar.GarageDoor.Geofence.getEventChangeAction(teslamateCar))

	teslamateCar.PrevGeofence = "not_home"
	teslamateCar.CurGeofence = "home"

	assert.Equal(t, ActionOpen, teslamateCar.GarageDoor.Geofence.getEventChangeAction(teslamateCar))
}

func Test_isInsidePolygonGeo(t *testing.T) {
	p := Point{
		Lat: 46.19292902096646,
		Lng: -123.79984989897177,
	}

	assert.Equal(t, false, isInsidePolygonGeo(p, polygonGeofence.Close))

	p = Point{
		Lat: 46.19243683948096,
		Lng: -123.80103692981524,
	}

	assert.Equal(t, true, isInsidePolygonGeo(p, polygonGeofence.Open))
}

func Test_getEventAction_Polygon(t *testing.T) {
	polygonCar.InsidePolyCloseGeo = true
	polygonCar.InsidePolyOpenGeo = true
	polygonCar.CurrentLocation.Lat = 46.19292902096646
	polygonCar.CurrentLocation.Lng = -123.79984989897177

	assert.Equal(t, ActionClose, polygonCar.GarageDoor.Geofence.getEventChangeAction(polygonCar))
	assert.Equal(t, false, polygonCar.InsidePolyCloseGeo)
	assert.Equal(t, true, polygonCar.InsidePolyOpenGeo)

	polygonCar.InsidePolyOpenGeo = false
	polygonCar.CurrentLocation.Lat = 46.19243683948096
	polygonCar.CurrentLocation.Lng = -123.80103692981524

	assert.Equal(t, ActionOpen, polygonCar.GarageDoor.Geofence.getEventChangeAction(polygonCar))
}

func Test_CheckCircularGeofence_Leaving(t *testing.T) {
	mockGdo := &mocks.GDO{}
	distanceCar.GarageDoor.Opener = mockGdo
	defer mockGdo.AssertExpectations(t)

	mockGdo.EXPECT().SetGarageDoor(ActionClose).Return(nil)

	distanceCar.CurDistance = 0
	distanceCar.CurrentLocation.Lat = distanceGeofence.Center.Lat + 10
	distanceCar.CurrentLocation.Lng = distanceGeofence.Center.Lng

	assert.Equal(t, checkGeofenceWrapper(distanceCar), true)
}

// if close is not defined, should not trigger any action
func Test_CheckCircularGeofence_Leaving_NoClose(t *testing.T) {
	mockGdo := &mocks.GDO{}
	distanceCar.GarageDoor.Opener = mockGdo
	defer mockGdo.AssertExpectations(t)

	prevCloseDistance := distanceGeofence.CloseDistance
	distanceGeofence.CloseDistance = 0

	distanceCar.CurDistance = 0
	distanceCar.CurrentLocation.Lat = distanceGeofence.Center.Lat + 10
	distanceCar.CurrentLocation.Lng = distanceGeofence.Center.Lng

	assert.Equal(t, checkGeofenceWrapper(distanceCar), true)
	distanceGeofence.CloseDistance = prevCloseDistance // restore settings
}

func Test_CheckCircularGeofence_Arriving(t *testing.T) {
	mockGdo := &mocks.GDO{}
	distanceCar.GarageDoor.Opener = mockGdo
	defer mockGdo.AssertExpectations(t)

	mockGdo.EXPECT().SetGarageDoor(ActionOpen).Return(nil)

	distanceCar.CurDistance = 100
	distanceCar.CurrentLocation.Lat = distanceGeofence.Center.Lat
	distanceCar.CurrentLocation.Lng = distanceGeofence.Center.Lng

	assert.Equal(t, checkGeofenceWrapper(distanceCar), true)
}

func Test_CheckCircularGeofence_LeaveThenArrive(t *testing.T) {
	mockGdo := &mocks.GDO{}
	distanceCar.GarageDoor.Opener = mockGdo
	defer mockGdo.AssertExpectations(t)

	// TEST 1 - Leaving home, garage close
	mockGdo.EXPECT().SetGarageDoor(ActionClose).Return(nil)

	distanceCar.CurDistance = 0
	distanceCar.CurrentLocation.Lat = distanceGeofence.Center.Lat + 10
	distanceCar.CurrentLocation.Lng = distanceGeofence.Center.Lng

	CheckGeofence(distanceCar)
	// wait for oplock to release to ensure goroutine within CheckGeofence function has completed
	for {
		if !distanceCar.GarageDoor.OpLock {
			break
		}
	}

	mockGdo.AssertExpectations(t) // midpoint check

	// TEST 2 - Arriving home, garage open
	mockGdo.EXPECT().SetGarageDoor(ActionOpen).Return(nil)

	distanceCar.CurrentLocation.Lat = distanceGeofence.Center.Lat
	distanceCar.CurrentLocation.Lng = distanceGeofence.Center.Lng

	assert.Equal(t, checkGeofenceWrapper(distanceCar), true)
}

func Test_CheckTeslamateGeofence_Leaving(t *testing.T) {
	mockGdo := &mocks.GDO{}
	teslamateCar.GarageDoor.Opener = mockGdo
	defer mockGdo.AssertExpectations(t)

	// TEST 1 - Leaving home, garage close
	mockGdo.EXPECT().SetGarageDoor(ActionClose).Return(nil)

	teslamateCar.PrevGeofence = "home"
	teslamateCar.CurGeofence = "not_home"

	assert.Equal(t, checkGeofenceWrapper(teslamateCar), true)
}

func Test_CheckTeslamateGeofence_Leaving_NoClose(t *testing.T) {
	mockGdo := &mocks.GDO{}
	teslamateCar.GarageDoor.Opener = mockGdo
	defer mockGdo.AssertExpectations(t)

	prevGeofenceClose := teslamateGeofence.Close
	teslamateGeofence.Close = TeslamateGeofenceTrigger{}

	teslamateCar.PrevGeofence = "home"
	teslamateCar.CurGeofence = "not_home"

	assert.Equal(t, checkGeofenceWrapper(teslamateCar), true)
	teslamateGeofence.Close = prevGeofenceClose // restore settings
}

func Test_CheckTeslamateGeofence_Arriving(t *testing.T) {
	mockGdo := &mocks.GDO{}
	teslamateCar.GarageDoor.Opener = mockGdo
	defer mockGdo.AssertExpectations(t)

	// TEST 1 - Arriving home, garage open
	mockGdo.EXPECT().SetGarageDoor(ActionOpen).Return(nil)

	teslamateCar.PrevGeofence = "not_home"
	teslamateCar.CurGeofence = "home"

	assert.Equal(t, checkGeofenceWrapper(teslamateCar), true)
}

func Test_CheckPolyGeofence_Leaving(t *testing.T) {
	mockGdo := &mocks.GDO{}
	polygonCar.GarageDoor.Opener = mockGdo
	defer mockGdo.AssertExpectations(t)

	// TEST 1 - Leaving home, garage close
	mockGdo.EXPECT().SetGarageDoor(ActionClose).Return(nil)

	polygonCar.InsidePolyCloseGeo = true
	polygonCar.InsidePolyOpenGeo = true
	polygonCar.CurrentLocation.Lat = 46.19292902096646
	polygonCar.CurrentLocation.Lng = -123.79984989897177

	assert.Equal(t, checkGeofenceWrapper(polygonCar), true)
}

func Test_CheckPolyGeofence_Arriving(t *testing.T) {
	mockGdo := &mocks.GDO{}
	polygonCar.GarageDoor.Opener = mockGdo
	defer mockGdo.AssertExpectations(t)

	// TEST 1 - Arriving home, garage open
	mockGdo.EXPECT().SetGarageDoor(ActionOpen).Return(nil)

	polygonCar.InsidePolyCloseGeo = false
	polygonCar.InsidePolyOpenGeo = false
	polygonCar.CurrentLocation.Lat = 46.19243683948096
	polygonCar.CurrentLocation.Lng = -123.80103692981524

	assert.Equal(t, checkGeofenceWrapper(polygonCar), true)
}

// if close is not defined, should not trigger any action
func Test_CheckPolyGeofence_Leaving_NoClose(t *testing.T) {
	mockGdo := &mocks.GDO{}
	polygonCar.GarageDoor.Opener = mockGdo
	defer mockGdo.AssertExpectations(t)

	prevCloseGeofence := polygonGeofence.Close
	polygonGeofence.Close = []Point{}

	polygonCar.InsidePolyCloseGeo = true
	polygonCar.InsidePolyOpenGeo = true
	polygonCar.CurrentLocation.Lat = 46.19243683948096
	polygonCar.CurrentLocation.Lng = -123.80103692981524

	assert.Equal(t, checkGeofenceWrapper(polygonCar), true)
	polygonGeofence.Close = prevCloseGeofence // restore settings
}

// runs CheckGeofence and waits for the internal goroutine to complete, signified by the release of oplock,
// with 100 ms timeout
func checkGeofenceWrapper(car *Car) bool {
	CheckGeofence(car)
	// wait for oplock to be released with a 100 ms timeout
	for i := 0; i < 10; i++ {
		if !car.GarageDoor.OpLock {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}
