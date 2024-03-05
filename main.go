package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

func main() {
	apiKey, ok := os.LookupEnv("CLAUDE_API_KEY")
	if !ok {
		panic(fmt.Errorf("Could not fetch api key"))
	}

	prompt := ""
	if len(os.Args) > 1 {
		prompt = os.Args[1]
	} else {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Prompt:")
		prompt, _ = reader.ReadString('\n')
		prompt = strings.TrimSpace(prompt)
	}

	payload := MessageBody{
		Model: "claude-3-opus-20240229",
		Messages: []Message{
			TextMessage{Role: "user", Content: prompt},
		},
		MaxTokens: 2000,
	}

	jsonpayload, err := json.Marshal(payload)
	if err != nil {
		panic("failed to marshal payload")
	}

	url := "https://api.anthropic.com/v1/messages"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonpayload))
	if err != nil {
		panic("failed to create request")
	}

	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(fmt.Errorf("failed to execute request: %v", err))
	}
	defer resp.Body.Close()
	// Check the response

	if resp.StatusCode != http.StatusOK {
		// Read the body of the non-OK response
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			panic("failed to read response body on non-OK status")
		}
		body := string(bodyBytes)
		// Print the non-OK status and response body for debugging purposes
		fmt.Printf("Received non-OK response status: %d\nResponse body: %s\n", resp.StatusCode, body)
		// Panic with a formatted error including the non-OK status
		panic(fmt.Errorf("received non-OK response status: %d", resp.StatusCode))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		// Handle error, maybe return or log
		fmt.Printf("Error reading response body: %v\n", err)
		return // or continue based on how you want to handle the error
	} // Close the response body when done
	defer resp.Body.Close()
	var apiResponse MessageResponse
	if err := json.Unmarshal(bodyBytes, &apiResponse); err != nil {
		// Handle error, maybe return or log
		fmt.Printf("Error unmarshalling response body: %v\n", err)
		return // or continue based on your error handling strategy
	} // Print the content field of the first choice in the response

	fmt.Println(apiResponse.Content[0])
	// fmt.Println(apiResponse.Usage.InputTokens)
	// fmt.Println(apiResponse.Usage.OutputTokens)

}
