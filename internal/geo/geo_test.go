package geo

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/brchri/tesla-geogdo/internal/gdo"
	"github.com/brchri/tesla-geogdo/internal/mocks"
	"github.com/stretchr/testify/assert"

	"github.com/brchri/tesla-geogdo/internal/util"
)

var (
	distanceTracker    *Tracker
	distanceGarageDoor *GarageDoor
	distanceGeofence   *CircularGeofence

	teslamateGarageDoor *GarageDoor
	teslamateTracker    *Tracker
	teslamateGeofence   *TeslamateGeofence

	polygonGarageDoor *GarageDoor
	polygonTracker    *Tracker
	polygonGeofence   *PolygonGeofence
)

func init() {
	util.LoadConfig(filepath.Join("..", "..", "examples", "config.multiple.yml"))
	InitializeGdoFunc = func(config map[string]interface{}) (gdo.GDO, error) { return &mocks.GDO{}, nil }
	ParseGarageDoorConfig()

	// used for testing events based on distance
	distanceGarageDoor = GarageDoors[0]
	distanceTracker = distanceGarageDoor.Trackers[0]
	distanceTracker.GarageDoor = distanceGarageDoor
	distanceGeofence, _ = distanceTracker.GarageDoor.Geofence.(*CircularGeofence) // type cast geofence interface

	// used for testing events based on teslamate geofence changes
	teslamateGarageDoor = GarageDoors[1]
	teslamateTracker = teslamateGarageDoor.Trackers[0]
	teslamateTracker.GarageDoor = teslamateGarageDoor
	teslamateGeofence, _ = teslamateTracker.GarageDoor.Geofence.(*TeslamateGeofence) // type cast geofence interface

	// used for testing events based on teslamate geofence changes
	polygonGarageDoor = GarageDoors[2]
	polygonTracker = polygonGarageDoor.Trackers[0]
	polygonTracker.GarageDoor = polygonGarageDoor
	polygonGeofence, _ = polygonTracker.GarageDoor.Geofence.(*PolygonGeofence) // type cast geofence interface

	util.Config.Global.OpCooldown = 0

	os.Setenv("GDO_SKIP_FLAP_DELAY", "true") // for testing, skip 1.5s delay after gdo ops meant to prevent spam from flapping
}

func Test_getEventChangeAction_Circular(t *testing.T) {
	distanceTracker.CurDistance = 0
	distanceTracker.CurrentLocation.Lat = distanceGeofence.Center.Lat + 10
	distanceTracker.CurrentLocation.Lng = distanceGeofence.Center.Lng

	assert.Equal(t, ActionClose, distanceTracker.GarageDoor.Geofence.getEventChangeAction(distanceTracker))
	assert.Greater(t, distanceTracker.CurDistance, distanceGeofence.CloseDistance)

	distanceTracker.CurrentLocation.Lat = distanceGeofence.Center.Lat

	assert.Equal(t, ActionOpen, distanceTracker.GarageDoor.Geofence.getEventChangeAction(distanceTracker))
	assert.Less(t, distanceTracker.CurDistance, distanceGeofence.OpenDistance)
}

func Test_getEventChangeAction_Teslamate(t *testing.T) {
	teslamateTracker.PrevGeofence = "home"
	teslamateTracker.CurGeofence = "not_home"

	assert.Equal(t, ActionClose, teslamateTracker.GarageDoor.Geofence.getEventChangeAction(teslamateTracker))

	teslamateTracker.PrevGeofence = "not_home"
	teslamateTracker.CurGeofence = "home"

	assert.Equal(t, ActionOpen, teslamateTracker.GarageDoor.Geofence.getEventChangeAction(teslamateTracker))
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
	polygonTracker.InsidePolyCloseGeo = true
	polygonTracker.InsidePolyOpenGeo = true
	polygonTracker.CurrentLocation.Lat = 46.19292902096646
	polygonTracker.CurrentLocation.Lng = -123.79984989897177

	assert.Equal(t, ActionClose, polygonTracker.GarageDoor.Geofence.getEventChangeAction(polygonTracker))
	assert.Equal(t, false, polygonTracker.InsidePolyCloseGeo)
	assert.Equal(t, true, polygonTracker.InsidePolyOpenGeo)

	polygonTracker.InsidePolyOpenGeo = false
	polygonTracker.CurrentLocation.Lat = 46.19243683948096
	polygonTracker.CurrentLocation.Lng = -123.80103692981524

	assert.Equal(t, ActionOpen, polygonTracker.GarageDoor.Geofence.getEventChangeAction(polygonTracker))
}

func Test_CheckCircularGeofence_Leaving(t *testing.T) {
	mockGdo := &mocks.GDO{}
	distanceTracker.GarageDoor.Opener = mockGdo
	defer mockGdo.AssertExpectations(t)

	mockGdo.EXPECT().SetGarageDoor(ActionClose).Return(nil)

	distanceTracker.CurDistance = 0
	distanceTracker.CurrentLocation.Lat = distanceGeofence.Center.Lat + 10
	distanceTracker.CurrentLocation.Lng = distanceGeofence.Center.Lng

	assert.Equal(t, checkGeofenceWrapper(distanceTracker), true)
}

// if close is not defined, should not trigger any action
func Test_CheckCircularGeofence_Leaving_NoClose(t *testing.T) {
	mockGdo := &mocks.GDO{}
	distanceTracker.GarageDoor.Opener = mockGdo
	defer mockGdo.AssertExpectations(t)

	prevCloseDistance := distanceGeofence.CloseDistance
	distanceGeofence.CloseDistance = 0

	distanceTracker.CurDistance = 0
	distanceTracker.CurrentLocation.Lat = distanceGeofence.Center.Lat + 10
	distanceTracker.CurrentLocation.Lng = distanceGeofence.Center.Lng

	assert.Equal(t, checkGeofenceWrapper(distanceTracker), true)
	distanceGeofence.CloseDistance = prevCloseDistance // restore settings
}

func Test_CheckCircularGeofence_Arriving(t *testing.T) {
	mockGdo := &mocks.GDO{}
	distanceTracker.GarageDoor.Opener = mockGdo
	defer mockGdo.AssertExpectations(t)

	mockGdo.EXPECT().SetGarageDoor(ActionOpen).Return(nil)

	distanceTracker.CurDistance = 100
	distanceTracker.CurrentLocation.Lat = distanceGeofence.Center.Lat
	distanceTracker.CurrentLocation.Lng = distanceGeofence.Center.Lng

	assert.Equal(t, checkGeofenceWrapper(distanceTracker), true)
}

func Test_CheckCircularGeofence_LeaveThenArrive(t *testing.T) {
	mockGdo := &mocks.GDO{}
	distanceTracker.GarageDoor.Opener = mockGdo
	defer mockGdo.AssertExpectations(t)

	// TEST 1 - Leaving home, garage close
	mockGdo.EXPECT().SetGarageDoor(ActionClose).Return(nil)

	distanceTracker.CurDistance = 0
	distanceTracker.CurrentLocation.Lat = distanceGeofence.Center.Lat + 10
	distanceTracker.CurrentLocation.Lng = distanceGeofence.Center.Lng

	CheckGeofence(distanceTracker)
	// wait for oplock to release to ensure goroutine within CheckGeofence function has completed
	for {
		if !distanceTracker.GarageDoor.OpLock {
			break
		}
	}

	mockGdo.AssertExpectations(t) // midpoint check

	// TEST 2 - Arriving home, garage open
	mockGdo.EXPECT().SetGarageDoor(ActionOpen).Return(nil)

	distanceTracker.CurrentLocation.Lat = distanceGeofence.Center.Lat
	distanceTracker.CurrentLocation.Lng = distanceGeofence.Center.Lng

	assert.Equal(t, checkGeofenceWrapper(distanceTracker), true)
}

func Test_CheckTeslamateGeofence_Leaving(t *testing.T) {
	mockGdo := &mocks.GDO{}
	teslamateTracker.GarageDoor.Opener = mockGdo
	defer mockGdo.AssertExpectations(t)

	// TEST 1 - Leaving home, garage close
	mockGdo.EXPECT().SetGarageDoor(ActionClose).Return(nil)

	teslamateTracker.PrevGeofence = "home"
	teslamateTracker.CurGeofence = "not_home"

	assert.Equal(t, checkGeofenceWrapper(teslamateTracker), true)
}

func Test_CheckTeslamateGeofence_Leaving_NoClose(t *testing.T) {
	mockGdo := &mocks.GDO{}
	teslamateTracker.GarageDoor.Opener = mockGdo
	defer mockGdo.AssertExpectations(t)

	prevGeofenceClose := teslamateGeofence.Close
	teslamateGeofence.Close = TeslamateGeofenceTrigger{}

	teslamateTracker.PrevGeofence = "home"
	teslamateTracker.CurGeofence = "not_home"

	assert.Equal(t, checkGeofenceWrapper(teslamateTracker), true)
	teslamateGeofence.Close = prevGeofenceClose // restore settings
}

func Test_CheckTeslamateGeofence_Arriving(t *testing.T) {
	mockGdo := &mocks.GDO{}
	teslamateTracker.GarageDoor.Opener = mockGdo
	defer mockGdo.AssertExpectations(t)

	// TEST 1 - Arriving home, garage open
	mockGdo.EXPECT().SetGarageDoor(ActionOpen).Return(nil)

	teslamateTracker.PrevGeofence = "not_home"
	teslamateTracker.CurGeofence = "home"

	assert.Equal(t, checkGeofenceWrapper(teslamateTracker), true)
}

func Test_CheckPolyGeofence_Leaving(t *testing.T) {
	mockGdo := &mocks.GDO{}
	polygonTracker.GarageDoor.Opener = mockGdo
	defer mockGdo.AssertExpectations(t)

	// TEST 1 - Leaving home, garage close
	mockGdo.EXPECT().SetGarageDoor(ActionClose).Return(nil)

	polygonTracker.InsidePolyCloseGeo = true
	polygonTracker.InsidePolyOpenGeo = true
	polygonTracker.CurrentLocation.Lat = 46.19292902096646
	polygonTracker.CurrentLocation.Lng = -123.79984989897177

	assert.Equal(t, checkGeofenceWrapper(polygonTracker), true)
}

func Test_CheckPolyGeofence_Arriving(t *testing.T) {
	mockGdo := &mocks.GDO{}
	polygonTracker.GarageDoor.Opener = mockGdo
	defer mockGdo.AssertExpectations(t)

	// TEST 1 - Arriving home, garage open
	mockGdo.EXPECT().SetGarageDoor(ActionOpen).Return(nil)

	polygonTracker.InsidePolyCloseGeo = false
	polygonTracker.InsidePolyOpenGeo = false
	polygonTracker.CurrentLocation.Lat = 46.19243683948096
	polygonTracker.CurrentLocation.Lng = -123.80103692981524

	assert.Equal(t, checkGeofenceWrapper(polygonTracker), true)
}

// if close is not defined, should not trigger any action
func Test_CheckPolyGeofence_Leaving_NoClose(t *testing.T) {
	mockGdo := &mocks.GDO{}
	polygonTracker.GarageDoor.Opener = mockGdo
	defer mockGdo.AssertExpectations(t)

	prevCloseGeofence := polygonGeofence.Close
	polygonGeofence.Close = []Point{}

	polygonTracker.InsidePolyCloseGeo = true
	polygonTracker.InsidePolyOpenGeo = true
	polygonTracker.CurrentLocation.Lat = 46.19243683948096
	polygonTracker.CurrentLocation.Lng = -123.80103692981524

	assert.Equal(t, checkGeofenceWrapper(polygonTracker), true)
	polygonGeofence.Close = prevCloseGeofence // restore settings
}

// runs CheckGeofence and waits for the internal goroutine to complete, signified by the release of oplock,
// with 100 ms timeout
func checkGeofenceWrapper(tracker *Tracker) bool {
	CheckGeofence(tracker)
	// wait for oplock to be released with a 100 ms timeout
	for i := 0; i < 10; i++ {
		if !tracker.GarageDoor.OpLock {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}
