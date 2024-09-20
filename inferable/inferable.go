// Package inferable provides a client for interacting with the Inferable API.
package inferable

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// Version of the inferable package
const Version = "0.1.0"

const (
    DefaultAPIEndpoint = "https://api.inferable.ai"
)

type Function struct {
    Func        interface{}
    Schema      json.RawMessage
    Description string
    Name        string
    Service     string // New field for service name
}

type FunctionRegistry struct {
    functions map[string]Function
}

type Inferable struct {
	client           *Client
	apiEndpoint      string
	apiSecret        string
	functionRegistry FunctionRegistry
}

func New(apiEndpoint, apiSecret string) (*Inferable, error) {
	if apiEndpoint == "" {
		apiEndpoint = DefaultAPIEndpoint
	}
	client := NewClient(apiEndpoint, apiSecret)
	return &Inferable{
		client:           client,
		apiEndpoint:      apiEndpoint,
		apiSecret:        apiSecret,
		functionRegistry: FunctionRegistry{functions: make(map[string]Function)},
	}, nil
}

func (i *Inferable) RegisterFunc(fn Function) error {
	if existing, exists := i.functionRegistry.functions[fn.Name]; exists {
		if existing.Service != fn.Service {
			return fmt.Errorf("function with name '%s' already registered for a different service", fn.Name)
		}
		return fmt.Errorf("function with name '%s' already registered for service '%s'", fn.Name, fn.Service)
	}
	i.functionRegistry.functions[fn.Name] = fn
	return nil
}

func (i *Inferable) FetchData(options FetchDataOptions) ([]byte, error) {
	data, err := i.client.FetchData(options)
	return []byte(data), err
}

func (i *Inferable) CallFunc(name string, args ...interface{}) ([]reflect.Value, error) {
	fn, exists := i.functionRegistry.functions[name]
	if !exists {
		return nil, fmt.Errorf("function with name '%s' not found", name)
	}

	// Get the reflect.Value of the function
	fnValue := reflect.ValueOf(fn.Func)

	// Check if the number of arguments is correct
	if len(args) != fnValue.Type().NumIn() {
		return nil, fmt.Errorf("invalid number of arguments for function '%s'", name)
	}

	// Prepare the arguments
	inArgs := make([]reflect.Value, len(args))
	for i, arg := range args {
		inArgs[i] = reflect.ValueOf(arg)
	}

	// Call the function
	return fnValue.Call(inArgs), nil
}
