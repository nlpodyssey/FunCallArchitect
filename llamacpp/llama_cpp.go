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

package llamacpp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"time"
)

// Config represents the configuration for the LLM endpoint
type Config struct {
	APIKey      string
	Model       string
	Endpoint    string
	Temperature float64
	TopP        float64
	MaxTokens   int
	UseGrammar  bool
	Timeout     time.Duration
}

// CompletionRequest represents a request to the LLM endpoint
type CompletionRequest struct {
	Model       string      `json:"model"`
	Messages    []Message   `json:"messages"`
	Temperature float64     `json:"temperature"`
	TopP        float64     `json:"top_p"`
	MaxTokens   int         `json:"max_tokens"`
	JsonSchema  interface{} `json:"json_schema,omitempty"`
	Grammar     string      `json:"grammar,omitempty"`
	Seed        int         `json:"seed"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// CompletionResponse represents a response from the LLM endpoint
type CompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type Client struct {
	config Config
	client *http.Client
}

func NewClient(c Config) *Client {
	return &Client{
		config: c,
		client: &http.Client{
			Timeout: c.Timeout,
		},
	}
}

func (c *Client) Complete(messages [][2]string, jsonSchema string) (string, error) {
	conversation := make([]Message, len(messages))
	for i, m := range messages {
		conversation[i] = Message{Role: m[0], Content: m[1]}
	}

	if jsonSchema == "" || !c.config.UseGrammar {
		return c.complete(conversation, "")
	}

	grammar, err := jsonSchemaToGrammar(jsonSchema)
	if err != nil {
		return "", fmt.Errorf("error converting JSON schema to grammar: %w", err)
	}
	return c.complete(conversation, grammar)
}

func (c *Client) complete(messages []Message, grammar string) (string, error) {
	response, err := c.getCompletionResponse(messages, grammar)
	if err != nil {
		log.Fatalf("Error getting completion response: %v", err)
	}
	return removeControlTokens(response.Choices[0].Message.Content), nil
}

func (c *Client) getCompletionResponse(messages []Message, grammar string) (*CompletionResponse, error) {
	requestBody := CompletionRequest{
		Model:       c.config.Model,
		Messages:    messages,
		Temperature: c.config.Temperature,
		TopP:        c.config.TopP,
		MaxTokens:   c.config.MaxTokens,
		Seed:        42,
	}

	if c.config.UseGrammar && grammar != "" {
		requestBody.Grammar = grammar
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("error marshalling JSON: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.config.Endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making API request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Error response: %s", body)
	}

	var completionResponse CompletionResponse
	err = json.Unmarshal(body, &completionResponse)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling JSON: %w", err)
	}

	return &completionResponse, nil
}

func removeControlTokens(content string) string {
	re := regexp.MustCompile(`<\|.*?\|>`)
	return re.ReplaceAllString(content, "")
}

func (c *Client) CreateEmbedding(text string) ([]float32, error) {
	jsonBody, _ := json.Marshal(map[string]interface{}{
		"input": text,
	})

	req, err := http.NewRequest(http.MethodPost, c.config.Endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making API request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Error response: %s", resp.Status)
	}

	embedding, err := parseEmbeddingResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("error parsing embedding response: %v", err)
	}

	return embedding, nil
}

func parseEmbeddingResponse(response *http.Response) ([]float32, error) {
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}
	defer response.Body.Close()

	var result struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling JSON: %w", err)
	}

	embedding := make([]float32, len(result.Data[0].Embedding))
	for i, v := range result.Data[0].Embedding {
		embedding[i] = float32(v)
	}

	return embedding, nil
}
