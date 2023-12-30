package homebridge

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/brchri/tesla-geogdo/internal/util"
	logger "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type (
	HomebridgeGdo interface {
		SetGarageDoor(string) error
		ProcessShutdown()
	}

	homebridgeGdo struct {
		Settings struct {
			Connection struct {
				Host          string `yaml:"host"`
				Port          int    `yaml:"port"`
				User          string `yaml:"user"`
				Pass          string `yaml:"pass"`
				UseTls        bool   `yaml:"use_tls"`
				SkipTlsVerify bool   `yaml:"skip_tls_verify"`
			} `yaml:"connection"`
			Timeout   int `yaml:"timeout"`
			Accessory struct {
				UniqueId        string `yaml:"unique_id"`
				Characteristics struct {
					Status  string `yaml:"status"`
					Command string `yaml:"command"`
					Values  struct {
						Open  interface{} `yaml:"open"`
						Close interface{} `yaml:"close"`
					} `yaml:"values"`
				} `yaml:"characteristics"`
			} `yaml:"accessory"`
		}
		authToken string
	}
)

const (
	defaultPort = 8581
)

func init() {
	logger.SetFormatter(&util.CustomFormatter{})
	logger.SetOutput(os.Stdout)
	if val, ok := os.LookupEnv("DEBUG"); ok && strings.ToLower(val) == "true" {
		logger.SetLevel(logger.DebugLevel)
	}
}

func Initialize(config map[string]interface{}) (HomebridgeGdo, error) {
	return NewHomebridgeGdo(config)
}

func NewHomebridgeGdo(config map[string]interface{}) (HomebridgeGdo, error) {
	var h *homebridgeGdo

	yamlData, err := yaml.Marshal(config)
	if err != nil {
		logger.Fatal("Failed to marhsal garage doors yaml object")
	}
	err = yaml.Unmarshal(yamlData, &h)
	if err != nil {
		logger.Fatal("Failed to unmarhsal garage doors yaml object")
	}

	// set port if not set explicitly in the config
	if h.Settings.Connection.Port == 0 {
		h.Settings.Connection.Port = defaultPort
	}

	if h.Settings.Timeout == 0 {
		h.Settings.Timeout = 30
	}

	return h, h.ValidateMinimumHttpSettings()
}

func (h *homebridgeGdo) ValidateMinimumHttpSettings() error {
	var errors []string
	if h.Settings.Connection.Host == "" {
		errors = append(errors, "missing homebridge host setting")
	}
	if h.Settings.Connection.User == "" {
		errors = append(errors, "missing homebridge user setting")
	}
	if h.Settings.Connection.Pass == "" {
		errors = append(errors, "missing homebridge password setting")
	}
	if h.Settings.Accessory.UniqueId == "" {
		errors = append(errors, "missing homebridge accessorry.uniqueid setting")
	}
	if h.Settings.Accessory.Characteristics.Command == "" {
		errors = append(errors, "missing homebridge accessorry.characteristics.command setting")
	}
	if h.Settings.Accessory.Characteristics.Values.Open == "" && h.Settings.Accessory.Characteristics.Values.Close == "" {
		errors = append(errors, "missing homebridge accessorry.characteristics.values.{open or close} setting")
	}

	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, "; "))
	}
	return nil
}

func (h *homebridgeGdo) SetGarageDoor(action string) error {
	logger.Debugf("Setting garage door target state: %s", action)
	err := h.login()
	if err != nil {
		return fmt.Errorf("unable to login, received error %v", err)
	}

	var desiredTargetState string
	var desiredStartState string

	if action == "open" {
		desiredTargetState = fmt.Sprintf("%v", h.Settings.Accessory.Characteristics.Values.Open)
		desiredStartState = fmt.Sprintf("%v", h.Settings.Accessory.Characteristics.Values.Close)
	} else if action == "close" {
		desiredTargetState = fmt.Sprintf("%v", h.Settings.Accessory.Characteristics.Values.Close)
		desiredStartState = fmt.Sprintf("%v", h.Settings.Accessory.Characteristics.Values.Open)
	}

	if h.Settings.Accessory.Characteristics.Status != "" {
		state, err := h.getDoorStatus()
		if err != nil {
			return fmt.Errorf("unable to get door status, received error %v", err)
		}
		if state != desiredStartState {
			logger.Warnf("Action and state mismatch: garage state is not valid for executing requested action; current state %s; requested action: %s", state, action)
			return nil
		}
	}

	endpoint := "/api/accessories/" + h.Settings.Accessory.UniqueId
	type reqBodyStruct struct {
		CharacteristicType string `json:"characteristicType"`
		Value              string `json:"value"`
	}
	reqBody := &reqBodyStruct{
		CharacteristicType: h.Settings.Accessory.Characteristics.Command,
		Value:              desiredTargetState,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + h.authToken,
	}
	_, err = h.ExecuteApiCall(endpoint, "PUT", string(body), headers)
	if err != nil {
		return fmt.Errorf("received error when executing api call to homebridge server: %v", err)
	}
	if h.Settings.Accessory.Characteristics.Status == "" {
		logger.Debug("request sent successfully, but no status characteristic defined, unable to determine if operation successful")
		return nil
	}

	// wait for timeout
	start := time.Now()
	for time.Since(start) < time.Duration(h.Settings.Timeout)*time.Second {
		state, err := h.getDoorStatus()
		if err != nil {
			logger.Debugf("Unable to get door state, received err: %v", err)
			logger.Debugf("Will keep trying until timeout expires")
		} else if state == desiredTargetState {
			logger.Infof("Garage door state has been set successfully: %s", desiredTargetState)
			return nil
		} else {
			logger.Debugf("Current opener state: %s", state)
		}
		time.Sleep(1 * time.Second)
	}

	return nil
}

func (h *homebridgeGdo) getDoorStatus() (string, error) {
	logger.Debug("getting door status")
	endpoint := "/api/accessories/" + h.Settings.Accessory.UniqueId
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + h.authToken,
	}
	rBody, err := h.ExecuteApiCall(endpoint, "GET", "", headers)
	if err != nil {
		return "", err
	}

	type respBody struct {
		Values map[string]interface{} `json:"values"`
	}

	rb := &respBody{}
	err = json.Unmarshal([]byte(rBody), &rb)
	if err != nil {
		return "", err
	}
	for k, v := range rb.Values {
		if k == h.Settings.Accessory.Characteristics.Status {
			logger.Debugf("received door status: %v", v)
			return fmt.Sprintf("%v", v), nil
		}
	}

	return "", fmt.Errorf("could not get door status")
}

func (h *homebridgeGdo) login() error {
	logger.Debug("logging into homebridge")
	type loginBody struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	lb := loginBody{
		Username: h.Settings.Connection.User,
		Password: h.Settings.Connection.Pass,
	}

	endpoint := "/api/auth/login"
	body, err := json.Marshal(lb)
	if err != nil {
		return fmt.Errorf("unable to marshal json for username and password, received error: %v", err)
	}

	headers := map[string]string{
		"Content-Type": "application/json",
	}

	rBody, err := h.ExecuteApiCall(endpoint, "POST", string(body), headers)
	if err != nil {
		return fmt.Errorf("received error when executing api call: %v", err)
	}

	type respBody struct {
		AccessToken string `json:"access_token"`
	}

	rb := &respBody{}
	err = json.Unmarshal([]byte(rBody), &rb)
	if err != nil {
		logger.Debug("login successful, access token retrieved")
		return nil
	}
	if rb.AccessToken == "" {
		return fmt.Errorf("unable to retrieve access token from Homebridge server")
	}
	h.authToken = rb.AccessToken

	return nil
}

func (h *homebridgeGdo) ExecuteApiCall(endpoint string, method string, body string, headers map[string]string) (respBody string, err error) {
	// build url api prefix
	urlPrefix := "http"
	if h.Settings.Connection.UseTls {
		urlPrefix += "s"
	}
	urlPrefix += fmt.Sprintf("://%s:%d", h.Settings.Connection.Host, h.Settings.Connection.Port)
	url := urlPrefix + endpoint

	req, err := http.NewRequest(method, url, bytes.NewBuffer([]byte(body)))
	if err != nil {
		return
	}
	for k, v := range headers {
		req.Header.Add(k, v)
	}

	client := &http.Client{}
	if h.Settings.Connection.UseTls && h.Settings.Connection.SkipTlsVerify {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	logger.Debug("executing api call:")
	logger.Debugf(" url: %s", url)
	logger.Debugf(" method: %s", method)
	logger.Debugf(" body: %s", body)

	// execute request
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("unable to send command to http endpoint, received err: %v", err)
	}
	defer resp.Body.Close()

	// check for 2xx response code
	if resp.StatusCode > 300 {
		return "", fmt.Errorf("received unexpected http status code: %s", resp.Status)
	}

	rBody, err := io.ReadAll(resp.Body)
	logger.Debug("api call successful")

	return string(rBody), err
}

// stubbed function as there's no need to process shutdown for homebridgeGdo
func (h *homebridgeGdo) ProcessShutdown() {}
