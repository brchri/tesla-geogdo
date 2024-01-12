package geo

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

type (
	// defines triggers for open and close action for teslamate geofences
	TeslamateGeofence struct {
		Close TeslamateGeofenceTrigger `yaml:"close_trigger,omitempty"` // garage will close when vehicle moves from `from` to `to`
		Open  TeslamateGeofenceTrigger `yaml:"open_trigger,omitempty"`  // garage will open when vehicle moves from `from` to `to`
	}

	// defines which teslamate defined geofence change will trigger an event, e.g. "home" to "not_home"
	TeslamateGeofenceTrigger struct {
		From string `yaml:"from,omitempty"`
		To   string `yaml:"to,omitempty"`
	}
)

// gets action based on if there was a relevant geofence event change
func (t *TeslamateGeofence) getEventChangeAction(tracker *Tracker) (action string) {
	if t.Close.IsTriggerDefined() &&
		tracker.PrevGeofence == t.Close.From &&
		tracker.CurGeofence == t.Close.To {
		action = ActionClose
	} else if t.Open.IsTriggerDefined() &&
		tracker.PrevGeofence == t.Open.From &&
		tracker.CurGeofence == t.Open.To {
		action = ActionOpen
	}
	return
}

func (t TeslamateGeofenceTrigger) IsTriggerDefined() bool {
	return t.From != "" && t.To != ""
}

func (t *TeslamateGeofence) parseSettings(config map[string]interface{}) error {
	yamlData, err := yaml.Marshal(config)
	var settings TeslamateGeofence
	if err != nil {
		return fmt.Errorf("failed to marhsal geofence yaml object, error: %v", err)
	}
	err = yaml.Unmarshal(yamlData, &settings)
	if err != nil {
		return fmt.Errorf("failed to unmarhsal geofence yaml object, error: %v", err)
	}
	*t = settings
	return nil
}
