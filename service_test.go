package inferable

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"bytes"
	"net/http"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterFunc(t *testing.T) {
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
	err := service.RegisterFunc(Function{
		Func:        testFunc,
		Name:        "TestFunc",
		Description: "Test function",
	})
	require.NoError(t, err)

	// Try to register the same function again
	err = service.RegisterFunc(Function{
		Func: testFunc,
		Name: "TestFunc",
	})
	assert.Error(t, err)

	// Try to register a function with invalid input
	invalidFunc := func(a, b int) int { return a + b }
	err = service.RegisterFunc(Function{
		Func: invalidFunc,
		Name: "InvalidFunc",
	})
	assert.Error(t, err)
}

func TestRegistrationAndConfig(t *testing.T) {
	// Load environment variables
	if os.Getenv("INFERABLE_API_SECRET") == "" {
		err := godotenv.Load("./.env")

		if err != nil {
			panic(err)
		}
	}

	apiSecret := os.Getenv("INFERABLE_API_SECRET")
	machineID := "random-machine-id"

	require.NotEmpty(t, apiSecret, "INFERABLE_API_SECRET is not set in .env")

	// Create a new Inferable instance
	i, err := New(InferableOptions{
		APIEndpoint: DefaultAPIEndpoint,
		APISecret:   apiSecret,
		MachineID:   machineID,
	})
	require.NoError(t, err)

	// Register a service
	service, err := i.RegisterService("TestService")
	require.NoError(t, err)

	// Register a test function
	type TestInput struct {
		A int `json:"a"`
		B int `json:"b"`
		C []struct {
			D int           `json:"d"`
			E string        `json:"e"`
			F []interface{} `json:"f"`
		} `json:"c"`
	}

	testFunc := func(input TestInput) int { return input.A + input.B }

	err = service.RegisterFunc(Function{
		Func:        testFunc,
		Name:        "TestFunc",
		Description: "Test function",
	})

	require.NoError(t, err)

	// Call Listen to trigger registration
	err = service.Start()
	require.NoError(t, err)

	// Get the config and check the details
	config := service.GetConfig()

	// Verify non-sensitive information
	assert.NotEmpty(t, config.QueueURL)
	assert.NotEmpty(t, config.Region)
	assert.True(t, config.Enabled)
	assert.True(t, config.Expiration.After(time.Now()))
}

func TestErrorneousRegistration(t *testing.T) {
	// Load environment variables
	if os.Getenv("INFERABLE_API_SECRET") == "" {
		err := godotenv.Load("./.env")

		if err != nil {
			panic(err)
		}
	}

	apiSecret := os.Getenv("INFERABLE_API_SECRET")
	machineID := "random-machine-id"

	require.NotEmpty(t, apiSecret, "INFERABLE_API_SECRET is not set in .env")

	// Create a new Inferable instance
	i, err := New(InferableOptions{
		APIEndpoint: os.Getenv("INFERABLE_API_ENDPOINT"),
		APISecret:   apiSecret,
		MachineID:   machineID,
	})
	require.NoError(t, err)

	// Register a service
	service, err := i.RegisterService("TestService")
	require.NoError(t, err)

	type F struct {
		G int `json:"g"`
	}

	// Register a test function
	type TestInput struct {
		A int `json:"a"`
		B int `json:"b"`
		C []struct {
			D int    `json:"d"`
			E string `json:"e"`
			F []F    `json:"f"`
		} `json:"c"`
	}

	testFunc := func(input TestInput) int { return input.A + input.B }

	err = service.RegisterFunc(Function{
		Func:        testFunc,
		Name:        "TestFunc",
		Description: "Test function",
	})

	require.ErrorContains(t, err, "schema for function 'TestFunc' contains a $ref to an external definition. this is currently not supported.")
}

func TestServiceStartAndReceiveMessage(t *testing.T) {
	if os.Getenv("INFERABLE_API_SECRET") == "" {
		err := godotenv.Load("./.env")
		require.NoError(t, err, "Error loading .env file")
	}

	machineID := os.Getenv("INFERABLE_MACHINE_ID")
	apiSecret := os.Getenv("INFERABLE_API_SECRET")
	clusterId := os.Getenv("INFERABLE_CLUSTER_ID")

	require.NotEmpty(t, apiSecret, "INFERABLE_API_SECRET is not set in .env")
	require.NotEmpty(t, clusterId, "INFERABLE_CLUSTER_ID is not set in .env")

	// Create a new Inferable instance
	i, err := New(InferableOptions{
		APIEndpoint: DefaultAPIEndpoint,
		APISecret:   apiSecret,
		MachineID:   machineID,
	})
	require.NoError(t, err)

	// Register a service
	service, err := i.RegisterService("TestService")
	require.NoError(t, err)

	// Register a test function
	type TestInput struct {
		Message string `json:"message"`
	}

	testFunc := func(input TestInput) string { return "Received: " + input.Message }

	err = service.RegisterFunc(Function{
		Func:        testFunc,
		Name:        "TestFunc",
		Description: "Test function",
	})
	require.NoError(t, err)

	// Start the service
	err = service.Start()
	require.NoError(t, err)

	// Ensure the service is stopped at the end of the test
	defer service.Stop()

	// Use executeJobSync to invoke the function
	testMessage := "Hello, SQS!"
	executeJobSyncURL := fmt.Sprintf("https://api.inferable.ai/clusters/%s/execute", clusterId)
	payload := map[string]interface{}{
		"service":  "TestService",
		"function": "TestFunc",
		"input": map[string]string{
			"message": testMessage,
		},
	}

	jsonPayload, err := json.Marshal(payload)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", executeJobSyncURL, bytes.NewBuffer(jsonPayload))
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiSecret)

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	// Check if the job was executed successfully
	require.Equal(t, "resolution", result["resultType"])
	require.Equal(t, "success", result["status"])
}
