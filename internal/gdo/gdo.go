package gdo

import (
	"errors"
	"fmt"

	"github.com/brchri/tesla-geogdo/internal/gdo/homeassistant"
	"github.com/brchri/tesla-geogdo/internal/gdo/homebridge"
	"github.com/brchri/tesla-geogdo/internal/gdo/http"
	"github.com/brchri/tesla-geogdo/internal/gdo/mqtt"
	"github.com/brchri/tesla-geogdo/internal/gdo/ratgdo"
)

type GDO interface {
	// set garage door action, e.g. `open` or `close`
	SetGarageDoor(action string) (err error)
	// process any required shutdown events, such as service disconnects
	ProcessShutdown()
}

func Initialize(config map[string]interface{}) (GDO, error) {
	typeValue, exists := config["type"]
	if !exists {
		return nil, errors.New("gdo type not defined")
	}

	switch typeValue {
	case "ratgdo":
		return ratgdo.Initialize(config)
	case "mqtt":
		return mqtt.Initialize(config)
	case "http":
		return http.Initialize(config)
	case "homeassistant":
		return homeassistant.Initialize(config)
	case "homebridge":
		return homebridge.Initialize(config)
	default:
		return nil, fmt.Errorf("gdo type %s not recognized", typeValue)
	}
}
