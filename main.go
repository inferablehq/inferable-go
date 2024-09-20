package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/inferablehq/inferable-go/inferable"
	"github.com/joho/godotenv"
)

func echo(input string) string {
	return input
}

func main() {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Get the API endpoint and secret from .env
	apiEndpoint := os.Getenv("INFERABLE_API_ENDPOINT")
	apiSecret := os.Getenv("INFERABLE_API_SECRET")

	if apiEndpoint == "" || apiSecret == "" {
		log.Fatal("INFERABLE_API_ENDPOINT or INFERABLE_API_SECRET not set in .env file")
	}

	// Create a new Inferable instance
	inferableInstance, err := inferable.New(apiEndpoint, apiSecret)
	if err != nil {
		log.Fatalf("Error creating Inferable instance: %v", err)
	}

	// Define the schema for the echo function
	echoSchema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"input": {
				"type": "string",
				"description": "The string to be echoed"
			}
		},
		"required": ["input"]
	}`)

	// Register the echo function
	err = inferableInstance.RegisterFunc(inferable.Function{
		Func:    echo,
		Schema:  echoSchema,
		Service: "echo",
		Name:    "echo", // Add the Name field
	})
	if err != nil {
		log.Fatalf("Error registering echo function: %v", err)
	}

	fmt.Println("Echo function registered successfully")

	// Call the echo function
	testInput := "Hello, Inferable!"
	result, err := inferableInstance.CallFunc("echo", testInput)
	if err != nil {
		log.Fatalf("Error calling echo function: %v", err)
	}

	// Assert that the result is correct
	if len(result) != 1 {
		log.Fatalf("Expected 1 return value, got %d", len(result))
	}

	returnedString := result[0].Interface().(string)
	if returnedString != testInput {
		log.Fatalf("Echo function returned incorrect result. Expected: %s, Got: %s", testInput, returnedString)
	}

	fmt.Printf("Echo function called successfully. Input: %s, Output: %s\n", testInput, returnedString)

	// Fetch data from the API
	data, err := inferableInstance.FetchData(inferable.FetchDataOptions{
		Path:   "/live",
		Method: "GET",
	})
	if err != nil {
		log.Fatalf("Error fetching data: %v", err)
	}

	// Print the result
	fmt.Printf("Response from %s:\n%s\n", apiEndpoint, data)
}