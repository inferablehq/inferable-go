// Package inferable provides a client for interacting with the Inferable API.
package inferable

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// Version of the inferable package
const Version = "0.1.1"

const (
	DefaultAPIEndpoint = "https://api.inferable.ai"
)

type FunctionRegistry struct {
	services map[string]*Service
}

type Inferable struct {
	client           *Client
	apiEndpoint      string
	apiSecret        string
	functionRegistry FunctionRegistry
	machineID        string // New field for machine ID
}

type InferableOptions struct {
	APIEndpoint string
	APISecret   string
	MachineID   string
}

func New(options InferableOptions) (*Inferable, error) {
	if options.APIEndpoint == "" {
		options.APIEndpoint = DefaultAPIEndpoint
	}
	client, err := NewClient(ClientOptions{
		Endpoint: options.APIEndpoint,
		Secret:   options.APISecret,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating client: %v", err)
	}

	machineID := options.MachineID
	if machineID == "" {
		machineID = generateMachineID(8)
	}

	return &Inferable{
		client:           client,
		apiEndpoint:      options.APIEndpoint,
		apiSecret:        options.APISecret,
		functionRegistry: FunctionRegistry{services: make(map[string]*Service)},
		machineID:        machineID,
	}, nil
}

// Convenience reference to a service with name 'default'.
func (i *Inferable) DefaultService() (*Service, error) {
	if _, exists := i.functionRegistry.services["default"]; exists {
		return i.functionRegistry.services["default"], nil
	}

	return i.RegisterService("default")
}

func (i *Inferable) RegisterService(serviceName string) (*Service, error) {
	if _, exists := i.functionRegistry.services[serviceName]; exists {
		return nil, fmt.Errorf("service with name '%s' already registered", serviceName)
	}
	service := &Service{
		Name:      serviceName,
		Functions: make(map[string]Function),
		inferable: i, // Set the reference to the Inferable instance
	}
	i.functionRegistry.services[serviceName] = service
	return service, nil
}

func (i *Inferable) CallFunc(serviceName, funcName string, args ...interface{}) ([]reflect.Value, error) {
	service, exists := i.functionRegistry.services[serviceName]
	if !exists {
		return nil, fmt.Errorf("service with name '%s' not found", serviceName)
	}

	fn, exists := service.Functions[funcName]
	if !exists {
		return nil, fmt.Errorf("function with name '%s' not found in service '%s'", funcName, serviceName)
	}

	// Get the reflect.Value of the function
	fnValue := reflect.ValueOf(fn.Func)

	// Check if the number of arguments is correct
	if len(args) != fnValue.Type().NumIn() {
		return nil, fmt.Errorf("invalid number of arguments for function '%s'", funcName)
	}

	// Prepare the arguments
	inArgs := make([]reflect.Value, len(args))
	for i, arg := range args {
		inArgs[i] = reflect.ValueOf(arg)
	}

	// Call the function
	return fnValue.Call(inArgs), nil
}

func (i *Inferable) ToJSONDefinition() ([]byte, error) {
	definition := make(map[string]interface{})

	for serviceName, service := range i.functionRegistry.services {
		serviceDef := make(map[string]interface{})
		functions := make([]map[string]interface{}, 0)

		for _, function := range service.Functions {
			funcDef := map[string]interface{}{
				"name":        function.Name,
				"description": function.Description,
				"schema":      function.schema,
			}
			functions = append(functions, funcDef)
		}

		serviceDef["service"] = serviceName
		serviceDef["functions"] = functions
		definition = serviceDef // We only need one service per definition
		break                   // We only process the first service
	}

	return json.MarshalIndent(definition, "", "  ")
}

func (i *Inferable) FetchData(options FetchDataOptions) ([]byte, error) {
	// Add default Content-Type header if not present
	if options.Headers == nil {
		options.Headers = make(map[string]string)
	}
	if _, exists := options.Headers["Content-Type"]; !exists && options.Body != "" {
		options.Headers["Content-Type"] = "application/json"
	}

	data, err := i.client.FetchData(options)
	return []byte(data), err
}

func (i *Inferable) GetMachineID() string {
	return i.machineID
}

func (i *Inferable) ServerOk() error {
	data, err := i.client.FetchData(FetchDataOptions{
		Path:   "/live",
		Method: "GET",
	})
	if err != nil {
		return fmt.Errorf("error fetching data from /live: %v", err)
	}

	var response struct {
		Status string `json:"status"`
	}

	// Convert string to []byte before unmarshaling
	if err := json.Unmarshal([]byte(data), &response); err != nil {
		return fmt.Errorf("error unmarshaling response: %v", err)
	}

	if response.Status != "ok" {
		return fmt.Errorf("unexpected status from /live: %s", response.Status)
	}

	return nil
}
