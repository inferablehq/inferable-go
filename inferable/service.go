package inferable

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"time"

	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/invopop/jsonschema"
)

type Service struct {
	Name      string
	Functions map[string]Function
	inferable *Inferable
	// Add new fields to store registration details
	queueURL    string
	region      string
	enabled     bool
	expiration  time.Time
	credentials struct {
		AccessKeyID     string
		SecretAccessKey string
		SessionToken    string
	}
	consumer *SQSConsumer
	ctx      context.Context
	cancel   context.CancelFunc
}

type Function struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Schema      interface{} `json:"schema,omitempty"`
	Config      interface{}
	Func        interface{}
}

func (s *Service) RegisterFunc(fn Function) error {
	if _, exists := s.Functions[fn.Name]; exists {
		return fmt.Errorf("function with name '%s' already registered for service '%s'", fn.Name, s.Name)
	}

	// Validate that the function has exactly one argument and it's a struct
	fnType := reflect.TypeOf(fn.Func)
	if fnType.NumIn() != 1 {
		return fmt.Errorf("function '%s' must have exactly one argument", fn.Name)
	}
	argType := fnType.In(0)
	if argType.Kind() != reflect.Struct {
		return fmt.Errorf("function '%s' argument must be a struct", fn.Name)
	}

	// Get the schema for the input struct
	reflector := jsonschema.Reflector{}
	schema := reflector.Reflect(reflect.New(argType).Interface())

	// Extract the relevant part of the schema
	defs, ok := schema.Definitions[argType.Name()]
	if !ok {
		return fmt.Errorf("failed to find schema definition for %s", argType.Name())
	}

	// Add the generated schema to the function
	fn.Schema = defs

	s.Functions[fn.Name] = fn
	return nil
}

func (s *Service) registerMachine() error {
	// Check if there are any registered functions
	if len(s.Functions) == 0 {
		return fmt.Errorf("cannot register service '%s': no functions registered", s.Name)
	}

	// Prepare the payload for registration
	payload := struct {
		Service   string `json:"service"`
		Functions []struct {
			Name        string `json:"name"`
			Description string `json:"description,omitempty"`
			Schema      string `json:"schema,omitempty"`
		} `json:"functions,omitempty"`
	}{
		Service: s.Name,
	}

	// Add registered functions to the payload
	for _, fn := range s.Functions {
		schemaJSON, err := json.Marshal(fn.Schema)
		if err != nil {
			return fmt.Errorf("failed to marshal schema for function '%s': %v", fn.Name, err)
		}

		payload.Functions = append(payload.Functions, struct {
			Name        string `json:"name"`
			Description string `json:"description,omitempty"`
			Schema      string `json:"schema,omitempty"`
		}{
			Name:        fn.Name,
			Description: fn.Description,
			Schema:      string(schemaJSON),
		})
	}

	// Marshal the payload to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %v", err)
	}

	// Prepare headers
	headers := map[string]string{
		"Authorization":          "Bearer " + s.inferable.apiSecret,
		"X-Machine-ID":           s.inferable.machineID,
		"X-Machine-SDK-Version":  Version,
		"X-Machine-SDK-Language": "go",
	}

	// Call the registerMachine endpoint
	options := FetchDataOptions{
		Path:    "/machines",
		Method:  "POST",
		Headers: headers,
		Body:    string(jsonPayload),
	}

	responseData, err := s.inferable.FetchData(options)
	if err != nil {
		return fmt.Errorf("failed to register machine: %v", err)
	}

	// Parse the response
	var response struct {
		QueueURL    string    `json:"queueUrl"`
		Region      string    `json:"region"`
		Enabled     bool      `json:"enabled"`
		Expiration  time.Time `json:"expiration"`
		Credentials struct {
			AccessKeyID     string `json:"accessKeyId"`
			SecretAccessKey string `json:"secretAccessKey"`
			SessionToken    string `json:"sessionToken"`
		} `json:"credentials"`
	}

	err = json.Unmarshal(responseData, &response)
	if err != nil {
		return fmt.Errorf("failed to parse registration response: %v", err)
	}

	// Store the registration details in the Service struct
	s.queueURL = response.QueueURL
	s.region = response.Region
	s.enabled = response.Enabled
	s.expiration = response.Expiration
	s.credentials.AccessKeyID = response.Credentials.AccessKeyID
	s.credentials.SecretAccessKey = response.Credentials.SecretAccessKey
	s.credentials.SessionToken = response.Credentials.SessionToken

	return nil
}

// Start initializes the service, registers the machine, and starts polling for messages
func (s *Service) Start() error {
	err := s.registerMachine()
	if err != nil {
		return fmt.Errorf("failed to register machine: %v", err)
	}

	// Create a new SQSConsumer with credentials
	consumer, err := NewSQSConsumer(
		s.region,
		s.queueURL,
		s.handleMessage,
		s.credentials.AccessKeyID,
		s.credentials.SecretAccessKey,
		s.credentials.SessionToken,
	)

	if err != nil {
		return fmt.Errorf("failed to create SQS consumer: %v", err)
	}

	s.consumer = consumer

	// Create a new context with cancellation
	s.ctx, s.cancel = context.WithCancel(context.Background())

	// Start polling for messages and handle potential errors
	go func() {
		if err := s.consumer.Start(s.ctx); err != nil {
			log.Printf("Error starting SQS consumer: %v", err)
			s.Stop() // Stop the service if there's an error starting the consumer
		}
	}()

	log.Printf("Service '%s' started and polling for messages", s.Name)
	return nil
}

// Stop stops the service and cancels the polling
func (s *Service) Stop() {
	if s.cancel != nil {
		s.cancel()
		log.Printf("Service '%s' stopped", s.Name)
	}
}

// handleMessage is a dummy message handler that just logs the received message
func (s *Service) handleMessage(msg *sqs.Message) error {
	log.Printf("Received message: %s", *msg.Body)

	// Define a struct to unmarshal the outer JSON structure
	var outerPayload struct {
		Value struct {
			ID         string `json:"id"`
			Service    string `json:"service"`
			TargetFn   string `json:"targetFn"`
			TargetArgs string `json:"targetArgs"` // Changed to string
		} `json:"value"`
	}

	// Unmarshal the message body into the outer payload struct
	if err := json.Unmarshal([]byte(*msg.Body), &outerPayload); err != nil {
		return fmt.Errorf("failed to unmarshal message body: %v", err)
	}

	// Call acknowledgeJob
	if err := s.acknowledgeJob(outerPayload.Value.ID); err != nil {
		log.Printf("Failed to acknowledge job: %v", err)
		// Continue processing the job even if acknowledgement fails
	}

	// Find the target function
	fn, ok := s.Functions[outerPayload.Value.TargetFn]
	if !ok {
		return fmt.Errorf("function not found: %s", outerPayload.Value.TargetFn)
	}

	// Unmarshal the target arguments string into a map
	var argsMap map[string]json.RawMessage
	if err := json.Unmarshal([]byte(outerPayload.Value.TargetArgs), &argsMap); err != nil {
		return fmt.Errorf("failed to unmarshal target arguments: %v", err)
	}

	// Extract the "value" field from the argsMap
	valueJSON, ok := argsMap["value"]
	if !ok {
		return fmt.Errorf("'value' field not found in target arguments")
	}

	// Create a new instance of the function's input type
	fnType := reflect.TypeOf(fn.Func)
	argType := fnType.In(0)
	argPtr := reflect.New(argType)

	// Unmarshal the value JSON into the function's input type
	if err := json.Unmarshal(valueJSON, argPtr.Interface()); err != nil {
		return fmt.Errorf("failed to unmarshal value into function argument: %v", err)
	}

	// Call the function with the unmarshaled argument
	fnValue := reflect.ValueOf(fn.Func)
	returnValues := fnValue.Call([]reflect.Value{argPtr.Elem()})

	log.Printf("Function '%s' called successfully", fn.Name)

	// Prepare the result
	result, err := s.prepareResult(returnValues)
	if err != nil {
		return fmt.Errorf("failed to prepare result: %v", err)
	}

	// Persist the job result
	if err := s.persistJobResult(outerPayload.Value.ID, result); err != nil {
		return fmt.Errorf("failed to persist job result: %v", err)
	}

	return nil
}

func (s *Service) prepareResult(returnValues []reflect.Value) (struct {
	Value string `json:"value"`
	Type  string `json:"type"`
}, error) {
	var result struct {
		Value string `json:"value"`
		Type  string `json:"type"`
	}

	if len(returnValues) > 0 {
		if errInterface, ok := returnValues[0].Interface().(error); ok {
			if errInterface != nil {
				result.Value = errInterface.Error()
				result.Type = "rejection"
			}
		} else {
			resultJSON, err := json.Marshal(returnValues[0].Interface())
			if err != nil {
				return result, fmt.Errorf("failed to marshal result: %v", err)
			}
			result.Value = string(resultJSON)
			result.Type = "resolution"
		}
	}

	return result, nil
}

func (s *Service) persistJobResult(jobID string, result struct {
	Value string `json:"value"`
	Type  string `json:"type"`
}) error {
	payload := struct {
		Result                string `json:"result"`
		ResultType            string `json:"resultType"`
		FunctionExecutionTime *int64 `json:"functionExecutionTime,omitempty"`
	}{
		Result:     result.Value,
		ResultType: result.Type,
		// You can add function execution time here if you measure it
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload for persistJobResult: %v", err)
	}

	headers := map[string]string{
		"Authorization":          "Bearer " + s.inferable.apiSecret,
		"X-Machine-ID":           s.inferable.machineID,
		"X-Machine-SDK-Version":  Version,
		"X-Machine-SDK-Language": "go",
	}

	options := FetchDataOptions{
		Path:    fmt.Sprintf("/jobs/%s/result", jobID),
		Method:  "POST",
		Headers: headers,
		Body:    string(payloadJSON),
	}

	_, err = s.inferable.FetchData(options)
	if err != nil {
		return fmt.Errorf("failed to persist job result: %v", err)
	}

	return nil
}

// Add the new acknowledgeJob function
func (s *Service) acknowledgeJob(jobID string) error {
	// Prepare headers
	headers := map[string]string{
		"Authorization":          "Bearer " + s.inferable.apiSecret,
		"X-Machine-ID":           s.inferable.machineID,
		"X-Machine-SDK-Version":  Version,
		"X-Machine-SDK-Language": "go",
	}

	// Call the acknowledgeJob endpoint
	options := FetchDataOptions{
		Path:    fmt.Sprintf("/jobs/%s", jobID),
		Method:  "PUT",
		Headers: headers,
	}

	_, err := s.inferable.FetchData(options)
	if err != nil {
		return fmt.Errorf("failed to acknowledge job: %v", err)
	}

	return nil
}

// Config represents the configuration of the service with obfuscated sensitive details
type Config struct {
	QueueURL    string    `json:"queueUrl"`
	Region      string    `json:"region"`
	Enabled     bool      `json:"enabled"`
	Expiration  time.Time `json:"expiration"`
	Credentials struct {
		AccessKeyID     string `json:"accessKeyId"`
		SecretAccessKey string `json:"secretAccessKey"`
		SessionToken    string `json:"sessionToken"`
	} `json:"credentials"`
}

// GetConfig returns the current configuration with obfuscated sensitive details
func (s *Service) GetConfig() Config {
	config := Config{
		QueueURL:   s.queueURL,
		Region:     s.region,
		Enabled:    s.enabled,
		Expiration: s.expiration,
	}

	return config
}
