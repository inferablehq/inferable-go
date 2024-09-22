package inferable

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
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
}

func (s *Service) RegisterFunc(fn Function) error {
	if _, exists := s.Functions[fn.Name]; exists {
		return fmt.Errorf("function with name '%s' already registered for service '%s'", fn.Name, s.Name)
	}
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
		payload.Functions = append(payload.Functions, struct {
			Name        string `json:"name"`
			Description string `json:"description,omitempty"`
			Schema      string `json:"schema,omitempty"`
		}{
			Name:        fn.Name,
			Description: fn.Description,
			Schema:      string(fn.Schema),
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

// Listen initializes the service, registers the machine, and stores the registration details
func (s *Service) Start() error {
	err := s.registerMachine()
	if err != nil {
		return fmt.Errorf("failed to register machine: %v", err)
	}

	// Start listening for messages (implement this later)
	// TODO: Implement message listening logic

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

	// Obfuscate sensitive credential information
	config.Credentials.AccessKeyID = obfuscateString(s.credentials.AccessKeyID)
	config.Credentials.SecretAccessKey = obfuscateString(s.credentials.SecretAccessKey)
	config.Credentials.SessionToken = obfuscateString(s.credentials.SessionToken)

	return config
}

// obfuscateString replaces all but the first and last 4 characters with asterisks
func obfuscateString(s string) string {
	if len(s) <= 8 {
		return strings.Repeat("*", len(s))
	}
	return s[:4] + strings.Repeat("*", len(s)-8) + s[len(s)-4:]
}
