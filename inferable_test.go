package inferable

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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

func TestCallFunc(t *testing.T) {
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

	result, err := i.CallFunc("TestService", "TestFunc", TestInput{A: 2, B: 3})
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
		Schema:      json.RawMessage(`{"input": {"a": "int", "b": "int"}, "output": "int"}`),
	})

	jsonDef, err := i.ToJSONDefinition()
	require.NoError(t, err)

	var definition map[string]interface{}
	err = json.Unmarshal(jsonDef, &definition)
	require.NoError(t, err)

	assert.Equal(t, "TestService", definition["service"])
	functions := definition["functions"].([]interface{})
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

// Add more tests as needed for other functions like FetchData, Start, etc.
