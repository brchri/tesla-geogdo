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

	geofenceGarageDoor *GarageDoor
	geofenceCar        *Car

	polygonGarageDoor *GarageDoor
	polygonCar        *Car
)

func init() {
	util.LoadConfig(filepath.Join("..", "..", "examples", "config.multiple.yml"))
	InitializeGdoFunc = func(config map[string]interface{}) (gdo.GDO, error) { return &mocks.GDO{}, nil }
	ParseGarageDoorConfig()

	// used for testing events based on distance
	distanceGarageDoor = GarageDoors[0]
	distanceCar = distanceGarageDoor.Cars[0]
	distanceCar.GarageDoor = distanceGarageDoor

	// used for testing events based on teslamate geofence changes
	geofenceGarageDoor = GarageDoors[1]
	geofenceCar = geofenceGarageDoor.Cars[0]
	geofenceCar.GarageDoor = geofenceGarageDoor

	// used for testing events based on teslamate geofence changes
	polygonGarageDoor = GarageDoors[2]
	polygonCar = polygonGarageDoor.Cars[0]
	polygonCar.GarageDoor = polygonGarageDoor

	util.Config.Global.OpCooldown = 0
}

func Test_getEventChangeAction_Circular(t *testing.T) {
	distanceCar.CurDistance = 0
	distanceCar.CurrentLocation.Lat = distanceCar.GarageDoor.CircularGeofence.Center.Lat + 10
	distanceCar.CurrentLocation.Lng = distanceCar.GarageDoor.CircularGeofence.Center.Lng

	assert.Equal(t, ActionClose, distanceCar.GarageDoor.Geofence.getEventChangeAction(distanceCar))
	assert.Greater(t, distanceCar.CurDistance, distanceCar.GarageDoor.CircularGeofence.CloseDistance)

	distanceCar.CurrentLocation.Lat = distanceCar.GarageDoor.CircularGeofence.Center.Lat

	assert.Equal(t, ActionOpen, distanceCar.GarageDoor.Geofence.getEventChangeAction(distanceCar))
	assert.Less(t, distanceCar.CurDistance, distanceCar.GarageDoor.CircularGeofence.OpenDistance)
}

func Test_getEventChangeAction_Teslamate(t *testing.T) {
	geofenceCar.PrevGeofence = "home"
	geofenceCar.CurGeofence = "not_home"

	assert.Equal(t, ActionClose, geofenceCar.GarageDoor.Geofence.getEventChangeAction(geofenceCar))

	geofenceCar.PrevGeofence = "not_home"
	geofenceCar.CurGeofence = "home"

	assert.Equal(t, ActionOpen, geofenceCar.GarageDoor.Geofence.getEventChangeAction(geofenceCar))
}

func Test_isInsidePolygonGeo(t *testing.T) {
	p := Point{
		Lat: 46.19292902096646,
		Lng: -123.79984989897177,
	}

	assert.Equal(t, false, isInsidePolygonGeo(p, polygonCar.GarageDoor.PolygonGeofence.Close))

	p = Point{
		Lat: 46.19243683948096,
		Lng: -123.80103692981524,
	}

	assert.Equal(t, true, isInsidePolygonGeo(p, polygonCar.GarageDoor.PolygonGeofence.Open))
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
	distanceCar.CurrentLocation.Lat = distanceGarageDoor.CircularGeofence.Center.Lat + 10
	distanceCar.CurrentLocation.Lng = distanceGarageDoor.CircularGeofence.Center.Lng

	assert.Equal(t, checkGeofenceWrapper(distanceCar), true)
}

// if close is not defined, should not trigger any action
func Test_CheckCircularGeofence_Leaving_NoClose(t *testing.T) {
	mockGdo := &mocks.GDO{}
	distanceCar.GarageDoor.Opener = mockGdo
	defer mockGdo.AssertExpectations(t)

	prevCloseDistance := distanceCar.GarageDoor.CircularGeofence.CloseDistance
	distanceCar.GarageDoor.CircularGeofence.CloseDistance = 0

	distanceCar.CurDistance = 0
	distanceCar.CurrentLocation.Lat = distanceGarageDoor.CircularGeofence.Center.Lat + 10
	distanceCar.CurrentLocation.Lng = distanceGarageDoor.CircularGeofence.Center.Lng

	assert.Equal(t, checkGeofenceWrapper(distanceCar), true)
	distanceCar.GarageDoor.CircularGeofence.CloseDistance = prevCloseDistance // restore settings
}

func Test_CheckCircularGeofence_Arriving(t *testing.T) {
	mockGdo := &mocks.GDO{}
	distanceCar.GarageDoor.Opener = mockGdo
	defer mockGdo.AssertExpectations(t)

	mockGdo.EXPECT().SetGarageDoor(ActionOpen).Return(nil)

	distanceCar.CurDistance = 100
	distanceCar.CurrentLocation.Lat = distanceGarageDoor.CircularGeofence.Center.Lat
	distanceCar.CurrentLocation.Lng = distanceGarageDoor.CircularGeofence.Center.Lng

	assert.Equal(t, checkGeofenceWrapper(distanceCar), true)
}

// // retry logic has been temporarily disabled, so this test is not needed until it's re-enabled
// // this is due to the myq api changes that need stabilizing so we don't retry and hit api rate limiting
// // func Test_CheckCircularGeofence_Arriving_LoggedIn_Retry(t *testing.T) {
// // 	myqSession := &mocks.MyqSessionInterface{}
// // 	myqSession.Test(t)
// // 	defer myqSession.AssertExpectations(t)
// // 	myqExec = myqSession

// // 	// TEST 1 - Arriving home, garage open
// // 	myqSession.EXPECT().DeviceState(mock.AnythingOfType("string")).Return(myq.StateClosed, nil).Times(3)
// // 	myqSession.EXPECT().SetDoorState(mock.AnythingOfType("string"), ActionOpen).Return(errors.New("some error")).Twice()
// // 	myqSession.EXPECT().SetDoorState(mock.AnythingOfType("string"), ActionOpen).Return(nil).Once()
// // 	myqSession.EXPECT().DeviceState(mock.AnythingOfType("string")).Return(myq.StateOpen, nil).Once()

// // 	distanceCar.CurDistance = 100
// // 	distanceCar.CurrentLocation.Lat = distanceGarageDoor.CircularGeofence.Center.Lat
// // 	distanceCar.CurrentLocation.Lng = distanceGarageDoor.CircularGeofence.Center.Lng

// // 	assert.Equal(t, checkGeofenceWrapper(distanceCar), true)
// // }

func Test_CheckCircularGeofence_LeaveThenArrive(t *testing.T) {
	mockGdo := &mocks.GDO{}
	distanceCar.GarageDoor.Opener = mockGdo
	defer mockGdo.AssertExpectations(t)

	// TEST 1 - Leaving home, garage close
	mockGdo.EXPECT().SetGarageDoor(ActionClose).Return(nil)

	distanceCar.CurDistance = 0
	distanceCar.CurrentLocation.Lat = distanceGarageDoor.CircularGeofence.Center.Lat + 10
	distanceCar.CurrentLocation.Lng = distanceGarageDoor.CircularGeofence.Center.Lng

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

	distanceCar.CurrentLocation.Lat = distanceGarageDoor.CircularGeofence.Center.Lat
	distanceCar.CurrentLocation.Lng = distanceGarageDoor.CircularGeofence.Center.Lng

	assert.Equal(t, checkGeofenceWrapper(distanceCar), true)
}

func Test_CheckTeslamateGeofence_Leaving(t *testing.T) {
	mockGdo := &mocks.GDO{}
	geofenceCar.GarageDoor.Opener = mockGdo
	defer mockGdo.AssertExpectations(t)

	// TEST 1 - Leaving home, garage close
	mockGdo.EXPECT().SetGarageDoor(ActionClose).Return(nil)

	geofenceCar.PrevGeofence = "home"
	geofenceCar.CurGeofence = "not_home"

	assert.Equal(t, checkGeofenceWrapper(geofenceCar), true)
}

func Test_CheckTeslamateGeofence_Leaving_NoClose(t *testing.T) {
	mockGdo := &mocks.GDO{}
	geofenceCar.GarageDoor.Opener = mockGdo
	defer mockGdo.AssertExpectations(t)

	prevGeofenceClose := geofenceCar.GarageDoor.TeslamateGeofence.Close
	geofenceCar.GarageDoor.TeslamateGeofence.Close = TeslamateGeofenceTrigger{}

	geofenceCar.PrevGeofence = "home"
	geofenceCar.CurGeofence = "not_home"

	assert.Equal(t, checkGeofenceWrapper(geofenceCar), true)
	geofenceCar.GarageDoor.TeslamateGeofence.Close = prevGeofenceClose // restore settings
}

func Test_CheckTeslamateGeofence_Arriving(t *testing.T) {
	mockGdo := &mocks.GDO{}
	geofenceCar.GarageDoor.Opener = mockGdo
	defer mockGdo.AssertExpectations(t)

	// TEST 1 - Arriving home, garage open
	mockGdo.EXPECT().SetGarageDoor(ActionOpen).Return(nil)

	geofenceCar.PrevGeofence = "not_home"
	geofenceCar.CurGeofence = "home"

	assert.Equal(t, checkGeofenceWrapper(geofenceCar), true)
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

	prevCloseGeofence := polygonCar.GarageDoor.PolygonGeofence.Close
	polygonCar.GarageDoor.PolygonGeofence.Close = []Point{}

	polygonCar.InsidePolyCloseGeo = true
	polygonCar.InsidePolyOpenGeo = true
	polygonCar.CurrentLocation.Lat = 46.19243683948096
	polygonCar.CurrentLocation.Lng = -123.80103692981524

	assert.Equal(t, checkGeofenceWrapper(polygonCar), true)
	polygonCar.GarageDoor.PolygonGeofence.Close = prevCloseGeofence // restore settings
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
