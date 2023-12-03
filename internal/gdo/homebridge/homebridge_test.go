package homebridge

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type httpRequestData struct {
	method  string
	path    string
	body    string
	headers []string
}

type statusResponseBody struct {
	AccessoryInformation map[string]interface{} `json:"accessoryInformation"`
	Values               struct {
		CurrentDoorState int `json:"CurrentDoorState"`
		TargetDoorState  int `json:"TargetDoorState"`
	} `json:"values"`
	UniqueId string `json:"uniqueId"`
}

var (
	testAccessoryId   = "1234"
	sampleAccessToken = "some_long_token"
	httpRequests      = []httpRequestData{}
	doorStateToReturn string

	sampleYaml = map[string]interface{}{
		"settings": map[string]interface{}{
			"connection": map[string]interface{}{
				"host": "localhost",
				"port": 80,
				"user": "test-user",
				"pass": "test-pass",
			},
			"accessory": map[string]interface{}{
				"unique_id": testAccessoryId,
				"characteristics": map[string]interface{}{
					"status":  "CurrentDoorState",
					"command": "TargetDoorState",
					"values": map[string]interface{}{
						"open":  0,
						"close": 1,
					},
				},
			},
		},
	}
)

func init() {
	os.Setenv("DEBUG", "true")
}

func Test_getDoorStatus(t *testing.T) {
	h, err := NewHomebridgeGdo(sampleYaml)
	assert.Equal(t, nil, err)
	if err != nil {
		return
	}
	hbGdo, ok := h.(*homebridgeGdo)
	if !ok {
		t.Error("returned type is not *homebridgeGdo")
	}

	mockServer := httptest.NewServer(http.HandlerFunc(mockServerHandler))
	defer mockServer.Close()
	re := regexp.MustCompile(`http[s]?:\/\/(.+):(.*)`)
	matches := re.FindStringSubmatch(mockServer.URL)
	hbGdo.Settings.Connection.Host = matches[1]
	serverPort, _ := strconv.ParseInt(matches[2], 10, 32)
	hbGdo.Settings.Connection.Port = int(serverPort)

	doorStateToReturn = "open"

	status, err := hbGdo.getDoorStatus()
	assert.Equal(t, nil, err)
	if err != nil {
		return
	}
	assert.Equal(t, "0", status)
}

func Test_SetGarageDoor_Open_NoStatus(t *testing.T) {
	h, err := NewHomebridgeGdo(sampleYaml)
	assert.Equal(t, nil, err)
	if err != nil {
		return
	}
	hbGdo, ok := h.(*homebridgeGdo)
	if !ok {
		t.Error("returned type is not *homebridgeGdo")
	}

	// create mock http server and extract host and port info
	mockServer := httptest.NewServer(http.HandlerFunc(mockServerHandler))
	defer mockServer.Close()
	re := regexp.MustCompile(`http[s]?:\/\/(.+):(.*)`)
	matches := re.FindStringSubmatch(mockServer.URL)
	hbGdo.Settings.Connection.Host = matches[1]
	serverPort, _ := strconv.ParseInt(matches[2], 10, 32)
	hbGdo.Settings.Connection.Port = int(serverPort)

	hbGdo.Settings.Accessory.Characteristics.Status = "" // disable status checks for this test
	hbGdo.SetGarageDoor("open")
	// check that we received some requests
	assert.LessOrEqual(t, 2, len(httpRequests))
	// check that final request was expected open params
	assert.Equal(t, "{\"characteristicType\":\"TargetDoorState\",\"value\":\"0\"}", httpRequests[len(httpRequests)-1].body)
	assert.Equal(t, "PUT", httpRequests[len(httpRequests)-1].method)
	assert.Equal(t, "/api/accessories/"+testAccessoryId, httpRequests[len(httpRequests)-1].path)
	assert.Equal(t, "Bearer "+sampleAccessToken, httpRequests[len(httpRequests)-1].headers[0])
}

// check SetGarageDoor with status checks
func Test_SetGarageDoor_Open_WithStatus(t *testing.T) {
	h, err := NewHomebridgeGdo(sampleYaml)
	assert.Equal(t, nil, err)
	if err != nil {
		return
	}
	hbGdo, ok := h.(*homebridgeGdo)
	if !ok {
		t.Error("returned type is not *homebridgeGdo")
	}

	// create mock http server and extract host and port info
	mockServer := httptest.NewServer(http.HandlerFunc(mockServerHandler))
	defer mockServer.Close()
	re := regexp.MustCompile(`http[s]?:\/\/(.+):(.*)`)
	matches := re.FindStringSubmatch(mockServer.URL)
	hbGdo.Settings.Connection.Host = matches[1]
	serverPort, _ := strconv.ParseInt(matches[2], 10, 32)
	hbGdo.Settings.Connection.Port = int(serverPort)

	// clear tracking vars
	httpRequests = []httpRequestData{}
	doorStateToReturn = "closed"

	// assert.Equal(t, nil, hbGdo.SetGarageDoor("open")) //test

	// execute SetGarageDoor in a goroutine so we can update the mocked door status
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		assert.Equal(t, nil, hbGdo.SetGarageDoor("open"))
	}()

	// watch for "door open" request with a timeout, and set the door state to "open"
	start := time.Now()
	for doorStateToReturn == "closed" && time.Since(start) < 5*time.Second {
		for _, v := range httpRequests {
			if v.path == "/api/accessories/"+testAccessoryId && v.method == "PUT" && v.body == "{\"characteristicType\":\"TargetDoorState\",\"value\":\"0\"}" {
				doorStateToReturn = "open"
				break
			}
		}
	}

	// wait for goroutine to finish
	wg.Wait()
}

func mockServerHandler(w http.ResponseWriter, r *http.Request) {
	bodyBytes, _ := io.ReadAll(r.Body)
	body := string(bodyBytes)

	httpRequest := httpRequestData{
		method:  r.Method,
		path:    r.URL.Path,
		body:    body,
		headers: r.Header.Values("Authorization"),
	}

	httpRequests = append(httpRequests, httpRequest)

	if r.Method == "GET" && r.URL.Path == "/api/accessories/"+testAccessoryId {
		respBody := &statusResponseBody{}
		if doorStateToReturn == "open" {
			respBody.Values.CurrentDoorState = 0
		} else {
			respBody.Values.CurrentDoorState = 1
		}
		jsonBody, err := json.Marshal(respBody)
		if err != nil {
			fmt.Fprint(w, err)
			return
		}
		fmt.Fprint(w, string(jsonBody))
		return
	}

	if r.Method == "POST" && r.URL.Path == "/api/auth/login" {
		fmt.Fprintf(w, `{"access_token": "%s"}`, sampleAccessToken)
		return
	}
}
