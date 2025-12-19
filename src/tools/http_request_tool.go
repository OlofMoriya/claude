package tools

import (
	"fmt"
	"io"
	"net/http"
	"owl/logger"
	"strings"
	"time"

	"github.com/fatih/color"
)

type HTTPRequestTool struct {
}

type HTTPRequestInput struct {
	URL     string
	Method  string
	Headers string
	Body    string
}

func (tool *HTTPRequestTool) Run(i map[string]string) (string, error) {

	logger.Screen(fmt.Sprintf("Asked to use http request with input %v", i), color.RGB(150, 150, 150))

	url, ok := i["URL"]
	if !ok || url == "" {
		return "", fmt.Errorf("URL is required")
	}

	method := strings.ToUpper(i["Method"])
	if method == "" {
		method = "GET"
	}

	// Validate method
	validMethods := map[string]bool{"GET": true, "POST": true, "PUT": true, "DELETE": true, "PATCH": true}
	if !validMethods[method] {
		return "", fmt.Errorf("Invalid HTTP method: %s", method)
	}

	// Create request
	var req *http.Request
	var err error

	body := i["Body"]
	if body != "" {
		req, err = http.NewRequest(method, url, strings.NewReader(body))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}

	if err != nil {
		return "", fmt.Errorf("Failed to create request: %s", err)
	}

	// Parse and add headers
	headers := i["Headers"]
	if headers != "" {
		for _, header := range strings.Split(headers, ";") {
			parts := strings.SplitN(header, ":", 2)
			if len(parts) == 2 {
				req.Header.Set(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
			}
		}
	}

	// Execute request with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	fmt.Printf("\nMaking %s request to: %s\n", method, url)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Request failed: %s", err)
	}
	defer resp.Body.Close()

	// Read response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("Failed to read response: %s", err)
	}

	result := fmt.Sprintf("Status: %d %s\n\nResponse:\n%s",
		resp.StatusCode,
		resp.Status,
		string(responseBody))

	return result, nil
}

func (tool *HTTPRequestTool) GetName() string {
	return "http_request"
}

func (tool *HTTPRequestTool) GetDefinition() Tool {
	return Tool{
		Name:        tool.GetName(),
		Description: "Makes HTTP requests to external APIs or services. Supports GET, POST, PUT, DELETE, and PATCH methods. Can include custom headers and request body. Useful for fetching data from APIs, webhooks, or integrating with external services.",

		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"URL": {
					Type:        "string",
					Description: "The full URL to make the request to. Required field. Example: https://api.example.com/data",
				},
				"Method": {
					Type:        "string",
					Description: "HTTP method to use: GET, POST, PUT, DELETE, or PATCH. Defaults to GET if not specified.",
				},
				"Headers": {
					Type:        "string",
					Description: "HTTP headers separated by semicolons. Format: 'HeaderName: Value; AnotherHeader: Value'. Example: 'Content-Type: application/json; Authorization: Bearer token123'",
				},
				"Body": {
					Type:        "string",
					Description: "Request body for POST, PUT, or PATCH requests. Typically JSON string.",
				},
			},
		},
	}
}

func init() {
	Register(&HTTPRequestTool{})
}
