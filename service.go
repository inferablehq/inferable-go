package inferable

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strings"
	"time"

	"github.com/invopop/jsonschema"
)

type Service struct {
	Name      string
	Functions map[string]Function
	inferable *Inferable
	clusterId string
	ctx       context.Context
	cancel    context.CancelFunc
}

type Function struct {
	Name        string
	Description string
	schema      interface{}
	Config      interface{}
	Func        interface{}
}

type CallMessage struct {
	Id       string      `json:"id"`
	Function string      `json:"function"`
	Input    interface{} `json:"input"`
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

	if schema == nil {
		return fmt.Errorf("failed to get schema for function '%s'", fn.Name)
	}

	// Extract the relevant part of the schema
	defs, ok := schema.Definitions[argType.Name()]
	if !ok {
		return fmt.Errorf("failed to find schema definition for %s", argType.Name())
	}

	defsString, err := json.Marshal(defs)
	if err != nil {
		return fmt.Errorf("failed to marshal schema for function '%s': %v", fn.Name, err)
	}

	if strings.Contains(string(defsString), "\"$ref\":\"#/$defs") {
		return fmt.Errorf("schema for function '%s' contains a $ref to an external definition. this is currently not supported. see https://go.inferable.ai/go-schema-limitation for details", fn.Name)
	}

	defs.AdditionalProperties = nil
	fn.schema = defs

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
		schemaJSON, err := json.Marshal(fn.schema)
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
		ClusterId string `json:"clusterId"`
	}

	err = json.Unmarshal(responseData, &response)
	if err != nil {
		return fmt.Errorf("failed to parse registration response: %v", err)
	}

	s.clusterId = response.ClusterId

	return nil
}

// Start initializes the service, registers the machine, and starts polling for messages
func (s *Service) Start() error {
	err := s.registerMachine()
	if err != nil {
		return fmt.Errorf("failed to register machine: %v", err)
	}

	s.ctx, s.cancel = context.WithCancel(context.Background())

	// Creat a run loop
	go func() {
		for {
			select {
			case <-s.ctx.Done():
				return
			default:
				// TODO: Retry-after
				// TODO: Error count
				err := s.poll()
				if err != nil {
					log.Printf("Failed to poll: %v", err)
				}
			}
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

func (s *Service) poll() error {
	headers := map[string]string{
		"Authorization":          "Bearer " + s.inferable.apiSecret,
		"X-Machine-ID":           s.inferable.machineID,
		"X-Machine-SDK-Version":  Version,
		"X-Machine-SDK-Language": "go",
	}

	options := FetchDataOptions{
		Path:    fmt.Sprintf("/clusters/%s/calls?acknowledge=true&service=%s&status=pending&limit=10", s.clusterId, s.Name),
		Method:  "GET",
		Headers: headers,
	}

	result, err := s.inferable.FetchData(options)
	if err != nil {
		return fmt.Errorf("failed to poll calls: %v", err)
	}

	parsed := []CallMessage{}

	err = json.Unmarshal(result, &parsed)
	if err != nil {
		return fmt.Errorf("failed to parse poll response: %v", err)
	}

	log.Printf("Polled for messages: %v", parsed)

	errors := []string{}
	for _, msg := range parsed {
		err := s.handleMessage(msg)
		if err != nil {
			errors = append(errors, err.Error())
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to handle messages: %v", errors)
	}

	return nil
}

func (s *Service) handleMessage(msg CallMessage) error {
	log.Printf("Received message: %s", msg.Id)

	// Find the target function
	fn, ok := s.Functions[msg.Function]
	if !ok {
		return fmt.Errorf("function not found: %s", msg.Function)
	}

	// Create a new instance of the function's input type
	fnType := reflect.TypeOf(fn.Func)
	argType := fnType.In(0)
	argPtr := reflect.New(argType)

	// // Unmarshal the value JSON into the function's input type
	// if err := json.Unmarshal(valueJSON, argPtr.Interface()); err != nil {
	// 	return fmt.Errorf("failed to unmarshal value into function argument: %v", err)
	// }

	// Call the function with the unmarshaled argument
	fnValue := reflect.ValueOf(fn.Func)
	returnValues := fnValue.Call([]reflect.Value{argPtr.Elem()})

	log.Printf("Function '%s' called successfully", fn.Name)

	start := time.Now()
	result := Result{
		Result:     returnValues[0].Interface(),
		ResultType: "resolution",
		Meta: ResultMetadata{
			FunctionExecutionTime: int64(time.Since(start).Milliseconds()),
		},
	}

	// Persist the job result
	if err := s.persistJobResult(msg.Id, result); err != nil {
		return fmt.Errorf("failed to persist job result: %v", err)
	}

	return nil
}

type ResultMetadata struct {
	FunctionExecutionTime int64 `json:"functionExecutionTime,omitempty"`
}

type Result struct {
	Result     interface{}    `json:"result"`
	ResultType string         `json:"resultType"`
	Meta       ResultMetadata `json:"meta"`
}

func (s *Service) persistJobResult(jobID string, result Result) error {
	payloadJSON, err := json.Marshal(result)
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
		Path:    fmt.Sprintf("/clusters/%s/calls/%s/result", s.clusterId, jobID),
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

func (s *Service) GetSchema() (map[string]interface{}, error) {
	if len(s.Functions) == 0 {
		return nil, fmt.Errorf("no functions registered for service '%s'", s.Name)
	}

	schema := make(map[string]interface{})

	for _, fn := range s.Functions {
		schema[fn.Name] = map[string]interface{}{
			"input": fn.schema,
			"name":  fn.Name,
		}
	}

	return schema, nil
}
