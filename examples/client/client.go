// Copyright 2024 The NLP Odyssey Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func main() {
	url := "http://localhost:8081/process"         // TODO: update URL
	message := "What's the weather like in Turin?" // TODO: update message

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBufferString(message))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}

	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Accept", "text/event-stream")

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Unexpected status code:", resp.StatusCode)
		return
	}

	if err := handleSSE(ctx, resp); err != nil {
		fmt.Println("Error handling SSE:", err)
	}
}

func handleSSE(ctx context.Context, resp *http.Response) error {
	scanner := bufio.NewScanner(resp.Body)
	var eventType string
	var eventData strings.Builder

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := scanner.Text()

		if line == "" {
			// Empty line indicates end of an event
			if eventType != "" && eventData.Len() > 0 {
				processEvent(eventType, eventData.String())
				eventType = ""
				eventData.Reset()
			}
		} else if strings.HasPrefix(line, "event:") {
			eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		} else if strings.HasPrefix(line, "data:") {
			eventData.WriteString(strings.TrimPrefix(line, "data:"))
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading SSE stream: %w", err)
	}
	return nil
}

func processEvent(eventType, eventData string) {
	var data map[string]any
	if err := json.Unmarshal([]byte(eventData), &data); err != nil {
		fmt.Printf("Error parsing JSON data for event type '%s': %v\n", eventType, err)
		return
	}

	message, ok := data["message"]
	if !ok {
		fmt.Printf("No 'message' field found in event data for event type '%s'\n", eventType)
		return
	}

	switch eventType {
	case "log":
		fmt.Printf("Log: %s\n", message)
	case "error":
		fmt.Printf("Error: %s\n", message)
	case "result":
		result, ok := message.(map[string]any)
		if !ok {
			fmt.Printf("Invalid 'message' field type for event type '%s'\n", eventType)
			return
		}
		funcCalls, ok := result["func_calls"]
		if !ok {
			fmt.Println("No 'func_calls' field found in result data")
			return
		}
		output, ok := result["output"]
		if !ok {
			fmt.Println("No 'output' field found in result data")
			return
		}

		fmt.Printf("Func Calls:\n%s\n\n", funcCalls)
		fmt.Printf("Output:\n%s\n", output)

	default:
		fmt.Printf("Unknown event type '%s': %s\n", eventType, message)
	}
}
