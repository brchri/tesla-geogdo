package geo

type (
	// defines triggers for open and close action for teslamate geofences
	TeslamateGeofence struct {
		Close TeslamateGeofenceTrigger `yaml:"close_trigger"` // garage will close when vehicle moves from `from` to `to`
		Open  TeslamateGeofenceTrigger `yaml:"open_trigger"`  // garage will open when vehicle moves from `from` to `to`
	}

	// defines which teslamate defined geofence change will trigger an event, e.g. "home" to "not_home"
	TeslamateGeofenceTrigger struct {
		From string `yaml:"from"`
		To   string `yaml:"to"`
	}
)

var teslamateMqttTopics = []string{"geofence"}

func (t *TeslamateGeofence) GetMqttTopics() []string {
	return teslamateMqttTopics
}

// gets action based on if there was a relevant geofence event change
func (t *TeslamateGeofence) getEventChangeAction(car *Car) (action string) {
	if car.GarageDoor.TeslamateGeofence.Close.IsTriggerDefined() &&
		car.PrevGeofence == car.GarageDoor.TeslamateGeofence.Close.From &&
		car.CurGeofence == car.GarageDoor.TeslamateGeofence.Close.To {
		action = ActionClose
	} else if car.GarageDoor.TeslamateGeofence.Open.IsTriggerDefined() &&
		car.PrevGeofence == car.GarageDoor.TeslamateGeofence.Open.From &&
		car.CurGeofence == car.GarageDoor.TeslamateGeofence.Open.To {
		action = ActionOpen
	}
	return
}

func (t TeslamateGeofenceTrigger) IsTriggerDefined() bool {
	return t.From != "" && t.To != ""
}
