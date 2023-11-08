package myq

// deprecating due to API changes and MyQ's general intention to shut down 3rd party integration
// leaving code base here for now for posterity

// import (
// 	"fmt"
// 	"io"
// 	"os"
// 	"strings"
// 	"time"

// 	"github.com/brchri/myq"
// 	"github.com/brchri/tesla-youq/internal/util"
// 	logger "github.com/sirupsen/logrus"
// )

// // interface that allows api calls to myq to be abstracted and mocked by testing functions
// type MyqSessionInterface interface {
// 	DeviceState(serialNumber string) (string, error)
// 	Login() error
// 	SetDoorState(serialNumber, action string) error
// 	SetUsername(string)
// 	SetPassword(string)
// 	GetToken() string
// 	SetToken(string)
// 	New()
// }

// // implements MyqSessionInterface interface but is only a wrapper for the actual myq package
// type MyqSessionWrapper struct {
// 	myqSession *myq.Session
// }

// func (m *MyqSessionWrapper) SetUsername(s string) {
// 	m.myqSession.Username = s
// }

// func (m *MyqSessionWrapper) SetPassword(s string) {
// 	m.myqSession.Password = s
// }

// func (m *MyqSessionWrapper) DeviceState(s string) (string, error) {
// 	return m.myqSession.DeviceState(s)
// }

// func (m *MyqSessionWrapper) Login() error {
// 	err := m.myqSession.Login()
// 	// cache token if requested
// 	if err == nil && util.Config.Global.CacheTokenFile != "" {
// 		file, fileErr := os.OpenFile(util.Config.Global.CacheTokenFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
// 		if fileErr != nil {
// 			logger.Infof("WARNING: Unable to write to cache file %s", util.Config.Global.CacheTokenFile)
// 		} else {
// 			defer file.Close()

// 			_, writeErr := file.WriteString(m.GetToken())
// 			if writeErr != nil {
// 				logger.Infof("WARNING: Unable to write to cache file %s", util.Config.Global.CacheTokenFile)
// 			}
// 		}
// 	}
// 	return err
// }

// func (m *MyqSessionWrapper) SetDoorState(serialNumber, action string) error {
// 	return m.myqSession.SetDoorState(serialNumber, action)
// }

// func (m *MyqSessionWrapper) New() {
// 	m.myqSession = &myq.Session{}
// }

// func (m *MyqSessionWrapper) GetToken() string {
// 	return m.myqSession.GetToken()
// }

// func (m *MyqSessionWrapper) SetToken(token string) {
// 	m.myqSession.SetToken(token)
// }

// var myqExec MyqSessionInterface // executes myq package commands

// func init() {
// 	myqExec = &MyqSessionWrapper{}
// 	myqExec.New()
// 	logger.SetFormatter(&util.CustomFormatter{})
// 	if val, ok := os.LookupEnv("DEBUG"); ok && strings.ToLower(val) == "true" {
// 		logger.SetLevel(logger.DebugLevel)
// 	}
// }

// func GetGarageDoorSerials(config util.ConfigStruct) error {
// 	s := &myq.Session{}
// 	s.Username = config.Global.MyQEmail
// 	s.Password = config.Global.MyQPass

// 	logger.Info("Acquiring MyQ session...")
// 	if err := s.Login(); err != nil {
// 		logger.Errorf("ERROR: %v", err)
// 		return err
// 	}
// 	logger.Info("Session acquired...")

// 	devices, err := s.Devices()
// 	if err != nil {
// 		logger.Infof("Could not get devices: %v", err)
// 		return err
// 	}
// 	for _, d := range devices {
// 		logger.Infof("Device Name: %v", d.Name)
// 		logger.Infof("Device State: %v", d.DoorState)
// 		logger.Infof("Device Type: %v", d.Type)
// 		logger.Infof("Device Serial: %v", d.SerialNumber)
// 		fmt.Println()
// 	}

// 	return nil
// }

// func SetGarageDoor(door GarageDoor, testing bool, action string) error {
// 	var desiredState string
// 	switch action {
// 	case myq.ActionOpen:
// 		desiredState = myq.StateOpen
// 	case myq.ActionClose:
// 		desiredState = myq.StateClosed
// 	}

// 	if testing {
// 		logger.Infof("TESTING flag set - Would attempt action %v", action)
// 		return nil
// 	}

// 	// check for cached token if we haven't retrieved it already
// 	if util.Config.Global.CacheTokenFile != "" && myqExec.GetToken() == "" {
// 		file, err := os.Open(util.Config.Global.CacheTokenFile)
// 		if err != nil {
// 			logger.Infof("WARNING: Unable to read token cache from %s", util.Config.Global.CacheTokenFile)
// 		} else {
// 			defer file.Close()

// 			data, err := io.ReadAll(file)
// 			if err != nil {
// 				logger.Infof("WARNING: Unable to read token cache from %s", util.Config.Global.CacheTokenFile)
// 			} else {
// 				myqExec.SetToken(string(data))
// 			}
// 		}
// 	}

// 	curState, err := myqExec.DeviceState(deviceSerial)
// 	if err != nil {
// 		// fetching device state may have failed due to invalid session token; try fresh login to resolve
// 		logger.Info("Acquiring MyQ session...")
// 		myqExec.New()
// 		myqExec.SetUsername(config.Global.MyQEmail)
// 		myqExec.SetPassword(config.Global.MyQPass)
// 		if err := myqExec.Login(); err != nil {
// 			logger.Infof("ERROR: %v", err)
// 			return err
// 		}
// 		logger.Info("Session acquired...")
// 		curState, err = myqExec.DeviceState(deviceSerial)
// 		if err != nil {
// 			logger.Infof("Couldn't get device state: %v", err)
// 			return err
// 		}
// 	}

// 	logger.Infof("Requested action: %v, Current state: %v", action, curState)
// 	if (action == myq.ActionOpen && curState == myq.StateClosed) || (action == myq.ActionClose && curState == myq.StateOpen) {
// 		logger.Infof("Attempting action: %v", action)
// 		err := myqExec.SetDoorState(deviceSerial, action)
// 		if err != nil {
// 			logger.Infof("Unable to set door state: %v", err)
// 			return err
// 		}
// 	} else {
// 		logger.Infof("Action and state mismatch: garage state is not valid for executing requested action")
// 		return nil
// 	}

// 	logger.Infof("Waiting for door to %s...", action)

// 	var currentState string
// 	deadline := time.Now().Add(60 * time.Second)
// 	for time.Now().Before(deadline) {
// 		state, err := myqExec.DeviceState(deviceSerial)
// 		if err != nil {
// 			return err
// 		}
// 		if state != currentState {
// 			if currentState != "" {
// 				logger.Infof("Door state changed to %s", state)
// 			}
// 			currentState = state
// 		}
// 		if currentState == desiredState {
// 			break
// 		}
// 		time.Sleep(5 * time.Second)
// 	}

// 	if currentState != desiredState {
// 		return fmt.Errorf("timed out waiting for door to be %s", desiredState)
// 	}

// 	return nil
// }
