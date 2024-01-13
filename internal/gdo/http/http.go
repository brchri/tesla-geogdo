package http

import (
	"bytes"
	"crypto/tls"
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
	HttpGdo interface {
		SetGarageDoor(string) error
		ProcessShutdown()
		// sets a parsing function that should be used to extract the status from an endpoint
		// http responses can be complex json blobs, and a simple `open` or `closed` is generally
		// expected for status; the parsing function, if set, will be used to extract that
		// simple status from more complex responses
		SetParseStatusResponseFunc(ParseStatusResponseFunc)
	}

	ParseStatusResponseFunc func(string) (string, error)

	httpGdo struct {
		Settings struct {
			Connection struct {
				Host          string `yaml:"host"`
				Port          int    `yaml:"port"`
				User          string `yaml:"user"`
				Pass          string `yaml:"pass"`
				UseTls        bool   `yaml:"use_tls,omitempty"`
				SkipTlsVerify bool   `yaml:"skip_tls_verify,omitempty"`
			} `yaml:"connection"`
			Status struct {
				Endpoint            string   `yaml:"endpoint,omitempty"`
				Headers             []string `yaml:"headers,omitempty"`
				ParseStatusResponse ParseStatusResponseFunc
			} `yaml:"status,omitempty"`
			Commands []Command `yaml:"commands"`
		} `yaml:"settings"`
		OpenerType   string `yaml:"type"` // name used by this module can be overridden by consuming modules, such as ratgdo, which is a wrapper for this package
		State        string // state of the garage door
		Availability string // if the garage door controller publishes an availability status (e.g. online), it will be stored here
		Obstruction  string // if the garage door controller publishes obstruction information, it will be stored here
	}

	Command struct {
		Name                string   `yaml:"name"` // e.g. `open` or `close`
		Endpoint            string   `yaml:"endpoint"`
		Headers             []string `yaml:"headers,omitempty"`
		HttpMethod          string   `yaml:"http_method"`
		Body                string   `yaml:"body,omitempty"`
		RequiredStartState  string   `yaml:"required_start_state,omitempty"`  // if set, garage door will not operate if current state does not equal this
		RequiredFinishState string   `yaml:"required_finish_state,omitempty"` // if set, garage door will monitor the door state compared to this value to determine success
		Timeout             int      `yaml:"timeout,omitempty"`               // time to wait for garage door to operate if monitored
	}
)

const (
	defaultHttpPort  = 80
	defaultHttpsPort = 443
)

func init() {
	logger.SetFormatter(&util.CustomFormatter{})
	logger.SetOutput(os.Stdout)
	if val, ok := os.LookupEnv("DEBUG"); ok && strings.ToLower(val) == "true" {
		logger.SetLevel(logger.DebugLevel)
	}
}

func Initialize(config map[string]interface{}) (HttpGdo, error) {
	return NewHttpGdo(config)
}

func NewHttpGdo(config map[string]interface{}) (HttpGdo, error) {
	var httpGdo *httpGdo

	yamlData, err := yaml.Marshal(config)
	if err != nil {
		logger.Fatal("Failed to marhsal garage doors yaml object")
	}
	err = yaml.Unmarshal(yamlData, &httpGdo)
	if err != nil {
		logger.Fatal("Failed to unmarhsal garage doors yaml object")
	}

	// set port if not set explicitly in the config
	if httpGdo.Settings.Connection.Port == 0 {
		if httpGdo.Settings.Connection.UseTls {
			httpGdo.Settings.Connection.Port = defaultHttpsPort
		} else {
			httpGdo.Settings.Connection.Port = defaultHttpPort
		}
	}

	// set command timeouts if not defined
	for k, c := range httpGdo.Settings.Commands {
		if c.Timeout == 0 {
			httpGdo.Settings.Commands[k].Timeout = 30
		}
	}

	return httpGdo, httpGdo.ValidateMinimumHttpSettings()
}

func (h *httpGdo) SetParseStatusResponseFunc(fn ParseStatusResponseFunc) {
	h.Settings.Status.ParseStatusResponse = fn
}

// will validate that the minimum mqtt settings are defined,
// will return nil if all settings validated successfully or
// an error if not
// also populates missing required settings with assumed defaults,
// such as port=1883
//
// validated settings are:
// host, commands[*].{Name,Payload,TopicSuffix}
func (h *httpGdo) ValidateMinimumHttpSettings() error {
	var errors []string
	if h.Settings.Connection.Host == "" {
		errors = append(errors, "missing http host setting")
	}
	if len(h.Settings.Commands) == 0 {
		errors = append(errors, "at least 1 command required to operate garage")
	}
	for i, c := range h.Settings.Commands {
		commandErrorFormat := "missing %s for command %d"
		if c.Name == "" {
			errors = append(errors, fmt.Sprintf(commandErrorFormat, "command name", i))
		}
		if c.Endpoint == "" {
			errors = append(errors, fmt.Sprintf(commandErrorFormat, "command endpoint", i))
		}
		if c.HttpMethod == "" {
			errors = append(errors, fmt.Sprintf(commandErrorFormat, "command http method", i))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, "; "))
	}
	return nil
}

func (h *httpGdo) SetGarageDoor(action string) error {
	// identify command based on action
	var command Command
	for _, v := range h.Settings.Commands {
		if action == v.Name {
			command = v
		}
	}
	if command.Name == "" {
		return fmt.Errorf("no command defined for action %s", action)
	}

	// validate required door state
	if command.RequiredStartState != "" && h.Settings.Status.Endpoint != "" {
		var err error
		h.State, err = h.getDoorStatus()
		if err == nil && h.Settings.Status.ParseStatusResponse != nil {
			h.State, err = h.Settings.Status.ParseStatusResponse(h.State)
		}
		if err != nil {
			return fmt.Errorf("unable to get door state, received err: %v", err)
		}
		if h.State != "" && h.State != command.RequiredStartState {
			logger.Warnf("Action and state mismatch: garage state is not valid for executing requested action; current state %s; requested action: %s", h.State, action)
			return nil
		}
	}

	if util.Config.Testing {
		logger.Infof("TESTING flag set - Would attempt action %v", action)
		return nil
	}

	// start building url and http client
	url := "http"
	if h.Settings.Connection.UseTls {
		url += "s"
	}
	url += fmt.Sprintf("://%s:%d%s", h.Settings.Connection.Host, h.Settings.Connection.Port, command.Endpoint)
	req, err := http.NewRequest(strings.ToUpper(command.HttpMethod), url, bytes.NewBuffer([]byte(command.Body)))
	if err != nil {
		return fmt.Errorf("unable to create http request, received err: %v", err)
	}

	addHeadersToReq(req, command.Headers)

	// set basic auth credentials if rqeuired
	if h.Settings.Connection.User != "" || h.Settings.Connection.Pass != "" {
		req.SetBasicAuth(h.Settings.Connection.User, h.Settings.Connection.Pass)
	}

	// initialize http client and configure tls settings if relevant
	client := &http.Client{}
	if h.Settings.Connection.UseTls && h.Settings.Connection.SkipTlsVerify {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	// execute request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("unable to send command to http endpoint, received err: %v", err)
	}
	defer resp.Body.Close()

	// check for 2xx response code
	if resp.StatusCode > 300 {
		return fmt.Errorf("received unexpected http status code: %s", resp.Status)
	}

	// if no required_finish_state or status.endpoint was defined, then just return that we successfully posted to the endpoint
	if command.RequiredFinishState == "" || h.Settings.Status.Endpoint == "" {
		logger.Infof("Garage door command `%s` has been sent to the http endpoint", action)
		return nil
	}

	// wait for timeout
	start := time.Now()
	for time.Since(start) < time.Duration(command.Timeout)*time.Second {
		h.State, err = h.getDoorStatus()
		if err == nil && h.Settings.Status.ParseStatusResponse != nil {
			h.State, err = h.Settings.Status.ParseStatusResponse(h.State)
		}
		if err != nil {
			logger.Debugf("Unable to get door state, received err: %v", err)
			logger.Debugf("Will keep trying until timeout expires")
		} else if h.State == command.RequiredFinishState {
			logger.Infof("Garage door state has been set successfully: %s", command.RequiredFinishState)
			return nil
		} else {
			logger.Debugf("Current opener state: %s", h.State)
		}
		time.Sleep(1 * time.Second)
	}

	// if we've hit this point, then we've timed out waiting for the garage to reach the requiredFinishState
	return fmt.Errorf("command sent to http endpoint, but timed out waiting for door to reach required_finish_state %s; current door state: %s", command.RequiredFinishState, h.State)
}

func (h *httpGdo) getDoorStatus() (string, error) {
	if h.Settings.Status.Endpoint == "" {
		// status endpoint not set, so just return empty string
		return "", nil
	}

	// start building url
	url := "http"
	if h.Settings.Connection.UseTls {
		url += "s"
	}
	url += fmt.Sprintf("://%s:%d%s", h.Settings.Connection.Host, h.Settings.Connection.Port, h.Settings.Status.Endpoint)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("unable to create http request, received err: %v", err)
	}

	if h.Settings.Connection.User != "" || h.Settings.Connection.Pass != "" {
		req.SetBasicAuth(h.Settings.Connection.User, h.Settings.Connection.Pass)
	}

	addHeadersToReq(req, h.Settings.Status.Headers)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("unable to request status from http endpoint, received err: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode > 300 {
		return "", fmt.Errorf("received unexpected http status code: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("unable to parse response body, received err: %v", err)
	}

	return string(body), nil

}

func addHeadersToReq(req *http.Request, headers []string) {
	for _, h := range headers {
		keyValPair := strings.SplitN(h, ":", 2)
		if len(keyValPair) != 2 {
			logger.Warnf("Unable to parse header %s", h)
			continue
		}
		req.Header.Add(keyValPair[0], keyValPair[1])
	}
}

// stubbed function for rquired interface, no shutdown routines required for this package
func (h *httpGdo) ProcessShutdown() {

}
