package inferable

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	i, err := New(InferableOptions{
		APIEndpoint: DefaultAPIEndpoint,
		APISecret:   "test-secret",
	})
	require.NoError(t, err)
	assert.Equal(t, DefaultAPIEndpoint, i.apiEndpoint)
	assert.Equal(t, "test-secret", i.apiSecret)
	assert.NotEmpty(t, i.machineID)
}

func TestRegisterService(t *testing.T) {
	i, _ := New(InferableOptions{
		APIEndpoint: DefaultAPIEndpoint,
		APISecret:   "test-secret",
	})
	service, err := i.RegisterService("TestService")
	require.NoError(t, err)
	assert.Equal(t, "TestService", service.Name)

	// Try to register the same service again
	_, err = i.RegisterService("TestService")
	assert.Error(t, err)
}

func TestRegisterDefaultService(t *testing.T) {
	i, err := New(InferableOptions{
		APIEndpoint: DefaultAPIEndpoint,
		APISecret:   "test-secret",
	})
	require.NoError(t, err)
	assert.Equal(t, "default", i.Default.Name)

	require.NoError(t, err)
}

func TestCallFunc(t *testing.T) {
	i, _ := New(InferableOptions{
		APIEndpoint: DefaultAPIEndpoint,
		APISecret:   "test-secret",
	})

	type TestInput struct {
		A int `json:"a"`
		B int `json:"b"`
	}

	testFunc := func(input TestInput) int { return input.A + input.B }
	i.Default.RegisterFunc(Function{
		Func: testFunc,
		Name: "TestFunc",
	})

	result, err := i.CallFunc("default", "TestFunc", TestInput{A: 2, B: 3})
	require.NoError(t, err)
	assert.Equal(t, 5, result[0].Interface())

	// Test calling non-existent function
	_, err = i.CallFunc("TestService", "NonExistentFunc")
	assert.Error(t, err)
}

func TestToJSONDefinition(t *testing.T) {
	i, _ := New(InferableOptions{
		APIEndpoint: DefaultAPIEndpoint,
		APISecret:   "test-secret",
	})
	service, _ := i.RegisterService("TestService")

	type TestInput struct {
		A int `json:"a"`
		B int `json:"b"`
	}

	testFunc := func(input TestInput) int { return input.A + input.B }
	service.RegisterFunc(Function{
		Func:        testFunc,
		Name:        "TestFunc",
		Description: "Test function",
	})

	jsonDef, err := i.ToJSONDefinition()
	require.NoError(t, err)

	var definitions []map[string]interface{}
	err = json.Unmarshal(jsonDef, &definitions)
	require.NoError(t, err)

	assert.Equal(t, "TestService", definitions[1]["service"])
	functions := definitions[1]["functions"].([]interface{})
	assert.Len(t, functions, 1)
	funcDef := functions[0].(map[string]interface{})
	assert.Equal(t, "TestFunc", funcDef["name"])
	assert.Equal(t, "Test function", funcDef["description"])
}

func TestServerOk(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/live" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "ok"}`))
		}
	}))
	defer server.Close()

	i, _ := New(InferableOptions{
		APIEndpoint: server.URL,
		APISecret:   "test-secret",
	})
	err := i.ServerOk()
	assert.NoError(t, err)
}

func TestGetMachineID(t *testing.T) {
	i, _ := New(InferableOptions{
		APIEndpoint: DefaultAPIEndpoint,
		APISecret:   "test-secret",
	})
	machineID := i.GetMachineID()
	assert.NotEmpty(t, machineID)

	// Check if the machine ID is persistent
	i2, _ := New(InferableOptions{
		APIEndpoint: DefaultAPIEndpoint,
		APISecret:   "test-secret",
	})
	assert.Equal(t, machineID, i2.GetMachineID())
}

func TestGetSchema(t *testing.T) {
	i, _ := New(InferableOptions{
		APIEndpoint: DefaultAPIEndpoint,
		APISecret:   "test-secret",
	})
	service, _ := i.RegisterService("TestService")

	type TestInput struct {
		A int `json:"a"`
		B int `json:"b"`
	}

	testFunc := func(input TestInput) int { return input.A + input.B }
	service.RegisterFunc(Function{
		Func: testFunc,
		Name: "TestFunc",
	})

	type TestInput2 struct {
		C struct {
			D int   `json:"d"`
			E []int `json:"e"`
		} `json:"c"`
	}

	testFunc2 := func(input TestInput2) int { return input.C.D * 2 }
	service.RegisterFunc(Function{
		Func: testFunc2,
		Name: "TestFunc2",
	})

	schema, err := service.GetSchema()
	require.NoError(t, err)
	assert.Equal(t, "TestFunc", schema["TestFunc"].(map[string]interface{})["name"])
	assert.Equal(t, "TestFunc2", schema["TestFunc2"].(map[string]interface{})["name"])

	// Marshal the schema to JSON and assert it as a string
	schemaJSON, err := json.Marshal(schema)
	require.NoError(t, err)
	assert.NotEmpty(t, string(schemaJSON))

	expectedJSON := `{
        "TestFunc": {
            "name": "TestFunc",
            "input": {
                "type": "object",
                "properties": {
                    "a": {"type": "integer"},
                    "b": {"type": "integer"}
                },
                "required": ["a", "b"]
            }
        },
        "TestFunc2": {
            "name": "TestFunc2",
            "input": {
                "type": "object",
                "properties": {
                    "c": {
                        "type": "object",
                        "properties": {
                            "d": {"type": "integer"},
                            "e": {"type": "array", "items": {"type": "integer"}}
                        },
                        "additionalProperties": false,
                        "required": ["d", "e"]
                    }
                },
                "required": ["c"]
            }
        }
    }`
	assert.JSONEq(t, expectedJSON, string(schemaJSON))
}

func TestPingCluster(t *testing.T) {
	pingCount := 0

	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if the request is a POST to /v2/ping
		if r.Method != "POST" || r.URL.Path != "/v2/ping" {
			t.Errorf("Expected POST request to /v2/ping, got %s request to %s", r.Method, r.URL.Path)
		}

		// Check the Content-Type header
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type header to be application/json, got %s", contentType)
		}

		// Read and parse the request body
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		// Check if the services field is present and is an array
		services, ok := body["services"].([]interface{})
		if !ok {
			t.Errorf("Expected 'services' field to be an array, got %T", body["services"])
		}

    if !reflect.DeepEqual(services, []interface{}{"default", "testService"}) {
			t.Errorf("Expected services array to contain ['default', 'testService'], got %v", services)

		}

		// Send a successful response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
		pingCount++
	}))
	defer server.Close()

	// Create a new Inferable instance with the mock server
	inferable, err := New(InferableOptions{
		APIEndpoint: server.URL,
		APISecret:   "test-secret",
	})
	if err != nil {
		t.Fatalf("Failed to create Inferable instance: %v", err)
	}

	// Register a test service
	_, err = inferable.RegisterService("testService")
	if err != nil {
		t.Fatalf("Failed to register service: %v", err)
	}

	// wait 2s. pingCluster should have been called at least once
	time.Sleep(2 * time.Second)
	assert.Greater(t, pingCount, 0)
}
