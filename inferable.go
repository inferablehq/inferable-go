// Package inferable provides a client for interacting with the Inferable API.
package inferable

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"

	"github.com/inferablehq/inferable-go/internal/client"
	"github.com/inferablehq/inferable-go/internal/util"
)

// Version of the inferable package
const Version = "0.1.13"

const (
	DefaultAPIEndpoint = "https://api.inferable.ai"
)

type functionRegistry struct {
	services map[string]*service
}

type Inferable struct {
	client           *client.Client
	apiEndpoint      string
	apiSecret        string
	functionRegistry functionRegistry
	machineID        string
	Default          *service
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
	client, err := client.NewClient(client.ClientOptions{
		Endpoint: options.APIEndpoint,
		Secret:   options.APISecret,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating client: %v", err)
	}

	machineID := options.MachineID
	if machineID == "" {
		machineID = util.GenerateMachineID(8)
	}

	inferable := &Inferable{
		client:           client,
		apiEndpoint:      options.APIEndpoint,
		apiSecret:        options.APISecret,
		functionRegistry: functionRegistry{services: make(map[string]*service)},
		machineID:        machineID,
	}

	// Automatically register the default service
	inferable.Default, err = inferable.RegisterService("default")
	if err != nil {
		return nil, fmt.Errorf("error registering default service: %v", err)
	}

	return inferable, nil
}

func (i *Inferable) RegisterService(serviceName string) (*service, error) {
	if _, exists := i.functionRegistry.services[serviceName]; exists {
		return nil, fmt.Errorf("service with name '%s' already registered", serviceName)
	}
	service := &service{
		Name:      serviceName,
		Functions: make(map[string]Function),
		inferable: i, // Set the reference to the Inferable instance
	}
	i.functionRegistry.services[serviceName] = service
	return service, nil
}

func (i *Inferable) callFunc(serviceName, funcName string, args ...interface{}) ([]reflect.Value, error) {
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

func (i *Inferable) toJSONDefinition() ([]byte, error) {
	definitions := make([]map[string]interface{}, 0)

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

		definitions = append(definitions, serviceDef)
	}

	return json.MarshalIndent(definitions, "", "  ")
}

func (i *Inferable) fetchData(options client.FetchDataOptions) ([]byte, http.Header, error, int) {
	// Add default Content-Type header if not present
	if options.Headers == nil {
		options.Headers = make(map[string]string)
	}
	if _, exists := options.Headers["Content-Type"]; !exists && options.Body != "" {
		options.Headers["Content-Type"] = "application/json"
	}

	data, headers, err, status:= i.client.FetchData(options)
	return []byte(data), headers, err, status
}

func (i *Inferable) serverOk() error {
	data, _, err, _ := i.client.FetchData(client.FetchDataOptions{
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
