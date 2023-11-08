package http

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/brchri/tesla-geogdo/internal/util"
	"github.com/stretchr/testify/assert"
)

type httpRequestData struct {
	method   string
	path     string
	body     string
	username string
	password string
}

var sampleYaml = map[string]interface{}{
	"settings": map[string]interface{}{
		"connection": map[string]interface{}{
			"host":            "localhost",
			"port":            80,
			"user":            "test-user",
			"pass":            "test-pass",
			"use_tls":         false,
			"skip_tls_verify": false,
		},
		"status": map[string]interface{}{
			"endpoint": "/status",
		},
		"commands": []map[string]interface{}{
			{
				"name":                  "open",
				"endpoint":              "/command",
				"http_method":           "post",
				"body":                  "{ \"command\": \"open\" }",
				"required_start_state":  "closed",
				"required_finish_state": "open",
				"timeout":               5,
			}, {
				"name":                  "close",
				"endpoint":              "/close",
				"http_method":           "post",
				"body":                  "",
				"required_start_state":  "open",
				"required_finish_state": "closed",
				"timeout":               5,
			},
		},
	},
}

var (
	httpRequests = []httpRequestData{}

	doorStateToReturn = "closed"
)

func Test_NewClient(t *testing.T) {
	// test with sample config defined above
	httpgdo, err := NewHttpGdo(sampleYaml)
	assert.Equal(t, nil, err)
	if err != nil {
		return
	}

	if h, ok := httpgdo.(*httpGdo); ok {
		assert.Equal(t, h.Settings.Connection.Host, "localhost")
		assert.Equal(t, h.Settings.Connection.Port, 80)
		assert.Equal(t, h.Settings.Status.Endpoint, "/status")
		assert.Equal(t, h.Settings.Commands[0].Name, "open")
		assert.Equal(t, h.Settings.Commands[1].Timeout, 5)
	} else {
		t.Error("returned type is not *httpGdo")
	}

	// test with sample config extracted from example config.yml file
	util.LoadConfig(filepath.Join("..", "..", "..", "examples", "config.polygon.http.yml"))
	door := *util.Config.GarageDoors[0]
	var openerConfig interface{}
	for k, v := range door {
		if k == "opener" {
			openerConfig = v
		}
	}
	if openerConfig == nil {
		t.Error("unable to parse config from garage door")
		return
	}
	config, ok := openerConfig.(map[string]interface{})
	if !ok {
		t.Error("unable to parse config from garage door")
		return
	}
	httpgdo, err = NewHttpGdo(config)
	assert.Equal(t, nil, err)

	if h, ok := httpgdo.(*httpGdo); ok {
		assert.Equal(t, h.Settings.Connection.Host, "localhost")
		assert.Equal(t, h.Settings.Connection.Port, 80)
		assert.Equal(t, h.Settings.Status.Endpoint, "/status")
		assert.Equal(t, h.Settings.Commands[0].Name, "open")
		assert.Equal(t, h.Settings.Commands[1].Timeout, 25)
	} else {
		t.Error("returned type is not *httpGdo")
	}
}

func mockServerHandler(w http.ResponseWriter, r *http.Request) {
	bodyBytes, _ := io.ReadAll(r.Body)
	body := string(bodyBytes)

	httpRequest := httpRequestData{
		method: r.Method,
		path:   r.URL.Path,
		body:   body,
	}
	httpRequest.username, httpRequest.password, _ = r.BasicAuth()

	httpRequests = append(httpRequests, httpRequest)

	if r.Method == "GET" && r.URL.Path == "/status" {
		fmt.Fprint(w, doorStateToReturn)
		return
	}

	if r.Method == "POST" && r.URL.Path == "/command" {
		fmt.Fprint(w, "")
	}
}

func Test_getDoorStatus(t *testing.T) {
	h, err := NewHttpGdo(sampleYaml)
	assert.Equal(t, nil, err)
	if err != nil {
		return
	}
	httpGdo, ok := h.(*httpGdo)
	if !ok {
		t.Error("returned type is not *httpGdo")
	}

	mockServer := httptest.NewServer(http.HandlerFunc(mockServerHandler))
	defer mockServer.Close()
	re := regexp.MustCompile(`http[s]?:\/\/(.+):(.*)`)
	matches := re.FindStringSubmatch(mockServer.URL)
	httpGdo.Settings.Connection.Host = matches[1]
	serverPort, _ := strconv.ParseInt(matches[2], 10, 32)
	httpGdo.Settings.Connection.Port = int(serverPort)

	doorStateToReturn = "open"

	state, err := httpGdo.getDoorStatus()
	assert.Equal(t, nil, err)
	if err != nil {
		return
	}
	assert.Equal(t, "open", state)

	doorStateToReturn = "closed"

	state, err = httpGdo.getDoorStatus()
	assert.Equal(t, nil, err)
	if err != nil {
		return
	}
	assert.Equal(t, "closed", state)
}

// check SetGarageDoor with no status checks
func Test_SetGarageDoor_Open_NoStatus(t *testing.T) {
	h, err := NewHttpGdo(sampleYaml)
	assert.Equal(t, nil, err)
	if err != nil {
		return
	}
	httpGdo, ok := h.(*httpGdo)
	if !ok {
		t.Error("returned type is not *httpGdo")
	}

	// create mock http server and extract host and port info
	mockServer := httptest.NewServer(http.HandlerFunc(mockServerHandler))
	defer mockServer.Close()
	re := regexp.MustCompile(`http[s]?:\/\/(.+):(.*)`)
	matches := re.FindStringSubmatch(mockServer.URL)
	httpGdo.Settings.Connection.Host = matches[1]
	serverPort, _ := strconv.ParseInt(matches[2], 10, 32)
	httpGdo.Settings.Connection.Port = int(serverPort)

	// clear status endpoint, as we're testing without it here
	httpGdo.Settings.Status.Endpoint = ""
	httpGdo.SetGarageDoor("open")
	// check that we received some requests
	assert.LessOrEqual(t, 1, len(httpRequests))
	// check that final request was expected open params
	assert.Equal(t, `{ "command": "open" }`, httpRequests[len(httpRequests)-1].body)
	assert.Equal(t, "POST", httpRequests[len(httpRequests)-1].method)
	assert.Equal(t, "/command", httpRequests[len(httpRequests)-1].path)
	assert.Equal(t, "test-user", httpRequests[len(httpRequests)-1].username)
	assert.Equal(t, "test-pass", httpRequests[len(httpRequests)-1].password)
}

// check SetGarageDoor with status checks
func Test_SetGarageDoor_Open_WithStatus(t *testing.T) {
	h, err := NewHttpGdo(sampleYaml)
	assert.Equal(t, nil, err)
	if err != nil {
		return
	}
	httpGdo, ok := h.(*httpGdo)
	if !ok {
		t.Error("returned type is not *httpGdo")
	}

	// create mock http server and extract host and port info
	mockServer := httptest.NewServer(http.HandlerFunc(mockServerHandler))
	defer mockServer.Close()
	re := regexp.MustCompile(`http[s]?:\/\/(.+):(.*)`)
	matches := re.FindStringSubmatch(mockServer.URL)
	httpGdo.Settings.Connection.Host = matches[1]
	serverPort, _ := strconv.ParseInt(matches[2], 10, 32)
	httpGdo.Settings.Connection.Port = int(serverPort)

	// clear tracking vars
	httpRequests = []httpRequestData{}
	doorStateToReturn = "closed"

	// execute SetGarageDoor in a goroutine so we can update the mocked door status
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		assert.Equal(t, nil, httpGdo.SetGarageDoor("open"))
	}()

	// watch for "door open" request with a timeout, and set the door state to "open"
	start := time.Now()
	for doorStateToReturn == "closed" && time.Since(start) < 5*time.Second {
		for _, v := range httpRequests {
			if v.path == "/command" && v.method == "POST" && v.body == `{ "command": "open" }` {
				doorStateToReturn = "open"
				break
			}
		}
	}

	// wait for goroutine to finish
	wg.Wait()
}

func Test_SetGarageDoor_Close_NoStatus(t *testing.T) {
	h, err := NewHttpGdo(sampleYaml)
	assert.Equal(t, nil, err)
	if err != nil {
		return
	}
	httpGdo, ok := h.(*httpGdo)
	if !ok {
		t.Error("returned type is not *httpGdo")
	}

	// create mock http server and extract host and port info
	mockServer := httptest.NewServer(http.HandlerFunc(mockServerHandler))
	defer mockServer.Close()
	re := regexp.MustCompile(`http[s]?:\/\/(.+):(.*)`)
	matches := re.FindStringSubmatch(mockServer.URL)
	httpGdo.Settings.Connection.Host = matches[1]
	serverPort, _ := strconv.ParseInt(matches[2], 10, 32)
	httpGdo.Settings.Connection.Port = int(serverPort)

	// clear status endpoint, as we're testing without it here
	httpGdo.Settings.Status.Endpoint = ""
	httpGdo.SetGarageDoor("close")
	// check that we received some requests
	assert.LessOrEqual(t, 1, len(httpRequests))
	// check that final request was expected open params
	assert.Equal(t, "", httpRequests[len(httpRequests)-1].body)
	assert.Equal(t, "POST", httpRequests[len(httpRequests)-1].method)
	assert.Equal(t, "/close", httpRequests[len(httpRequests)-1].path)
}
