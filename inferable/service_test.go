package inferable

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterFunc(t *testing.T) {
	i, _ := New("", "test-secret")
	service, _ := i.RegisterService("TestService")

	testFunc := func(a, b int) int { return a + b }
	err := service.RegisterFunc(Function{
		Func:        testFunc,
		Name:        "TestFunc",
		Description: "Test function",
		Schema:      json.RawMessage(`{"input": {"a": "int", "b": "int"}, "output": "int"}`),
	})
	require.NoError(t, err)

	// Try to register the same function again
	err = service.RegisterFunc(Function{
		Func: testFunc,
		Name: "TestFunc",
	})
	assert.Error(t, err)
}

func TestRegistrationAndConfig(t *testing.T) {
	// Load environment variables
	err := godotenv.Load("../.env")
	require.NoError(t, err, "Error loading .env file")

	machineID := os.Getenv("INFERABLE_MACHINE_ID")
	apiSecret := os.Getenv("INFERABLE_API_SECRET")
	require.NotEmpty(t, apiSecret, "INFERABLE_API_SECRET is not set in .env")

	// Create a new Inferable instance
	i, err := New(machineID, apiSecret)
	require.NoError(t, err)

	// Register a service
	service, err := i.RegisterService("TestService")
	require.NoError(t, err)

	// Register a test function
	testFunc := func(a, b int) int { return a + b }
	err = service.RegisterFunc(Function{
		Func:        testFunc,
		Name:        "TestFunc",
		Description: "Test function",
		Schema:      json.RawMessage(`{"input": {"a": "int", "b": "int"}, "output": "int"}`),
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

	// Check that sensitive information is obfuscated
	assert.Regexp(t, `^[A-Z0-9]{4}\*+[A-Z0-9]{4}$`, config.Credentials.AccessKeyID)
	assert.Regexp(t, `^[A-Za-z0-9]{4}\*+[A-Za-z0-9]{4}$`, config.Credentials.SecretAccessKey)
	assert.Regexp(t, `^[A-Za-z0-9]{4}\*+[A-Za-z0-9]{4}$`, config.Credentials.SessionToken)

	// Verify that we can't access the raw credentials through the config
	assert.NotEqual(t, service.credentials.AccessKeyID, config.Credentials.AccessKeyID)
	assert.NotEqual(t, service.credentials.SecretAccessKey, config.Credentials.SecretAccessKey)
	assert.NotEqual(t, service.credentials.SessionToken, config.Credentials.SessionToken)

	// Verify that the raw credentials are not empty in the service struct
	assert.NotEmpty(t, service.credentials.AccessKeyID)
	assert.NotEmpty(t, service.credentials.SecretAccessKey)
	assert.NotEmpty(t, service.credentials.SessionToken)
}
