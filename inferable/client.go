package inferable

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

// Client represents an Inferable API client
type Client struct {
	endpoint   string
	secret     string
	httpClient *http.Client
}

// NewClient creates a new Inferable API client
func NewClient(endpoint, secret string) *Client {
	return &Client{
		endpoint:   endpoint,
		secret:     secret,
		httpClient: &http.Client{},
	}
}

type FetchDataOptions struct {
	Path        string
	Headers     map[string]string
	QueryParams map[string]string
	Body        string
	Method      string
}

// FetchData makes a request to the Inferable API and returns the response
func (c *Client) FetchData(options FetchDataOptions) (string, error) {
	fullURL := fmt.Sprintf("%s%s", c.endpoint, options.Path)
	req, err := http.NewRequest(options.Method, fullURL, strings.NewReader(options.Body))
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.secret)

	// Add custom headers
	for key, value := range options.Headers {
		req.Header.Set(key, value)
	}

	// Add query parameters
	q := req.URL.Query()
	for key, value := range options.QueryParams {
		q.Add(key, value)
	}
	req.URL.RawQuery = q.Encode()

	// Set Content-Type header if body is not empty
	if options.Body != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %v", err)
	}

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("API error: %s (status code: %d)", string(body), resp.StatusCode)
	}

	return string(body), nil
}
