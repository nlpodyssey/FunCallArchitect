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

package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/nlpodyssey/funcallarchitect/execution"
	"github.com/nlpodyssey/funcallarchitect/llm"
	"github.com/nlpodyssey/funcallarchitect/parser"
	"github.com/nlpodyssey/funcallarchitect/progress"
	"github.com/nlpodyssey/funcallarchitect/prompt"
	"github.com/nlpodyssey/funcallarchitect/tools"
)

const UnprocessableRequestPrompt = "Unable to process this request. Please rephrase or provide a different query."

type Tools interface {
	RegisterWith(ec *execution.Orchestrator) error
	AvailableTools() *tools.ToolSet
}

type ProcessingResult struct {
	Execution *execution.Result
}

func UnprocessableRequestExecutions() *execution.Result {
	return &execution.Result{
		FuncCalls: []*execution.ExecutedFuncCall{
			{
				Name:    "__builtin__.unprocessable_request",
				Purpose: "Return a response for an unprocessable request",
				Args:    nil,
				Result: execution.FuncResult{
					Present: false,
					FormatFunc: func() (string, error) {
						return UnprocessableRequestPrompt, nil
					},
				},
			},
		},
	}
}

// RequestHandlerConfig holds the resources for the RequestHandler
type RequestHandlerConfig struct {
	Logger               *log.Logger
	LLMClient            llm.Completer
	Tools                Tools
	Timeout              time.Duration
	EnableConcurrentExec bool

	AlterUserRequest func(string) string
	AlterResult      func(result *ProcessingResult) error
}

// RequestHandler represents a generic agent that can interact with a set of tools
type RequestHandler struct {
	config       RequestHandlerConfig
	orchestrator *execution.Orchestrator
}

// NewRequestHandler creates a new RequestHandler instance
func NewRequestHandler(config RequestHandlerConfig) (*RequestHandler, error) {
	if config.Logger == nil {
		config.Logger = log.New(log.Writer(), "", log.Ldate|log.Ltime|log.Lshortfile)
	}

	ec := execution.NewOrchestrator(config.Logger, config.Timeout, config.EnableConcurrentExec, config.Tools.AvailableTools())

	agent := &RequestHandler{
		config:       config,
		orchestrator: ec,
	}

	if err := config.Tools.RegisterWith(ec); err != nil {
		agent.config.Logger.Printf("Failed to register tools: %v", err)
		return nil, fmt.Errorf("failed to register tools: %w", err)
	}

	return agent, nil
}

// ProcessUserRequest handles the user's request and returns the processing result
func (a *RequestHandler) ProcessUserRequest(ctx context.Context, message string, progress progress.Stream) (*ProcessingResult, error) {
	progress.Send("Processing user request...")

	if a.config.AlterUserRequest != nil {
		a.config.Logger.Printf("Original message: %s", message)
		message = a.config.AlterUserRequest(message)
		a.config.Logger.Printf("Altered message: %s", message)
	}

	funcCalls, err := a.generateFunctionCalls(ctx, message, progress)
	if err != nil {
		return nil, fmt.Errorf("error generating function calls: %w", err)
	}

	funcCalls, err = a.evaluateFuncCallsConsistency(message, funcCalls, progress)
	if err != nil {
		return nil, fmt.Errorf("error evaluating function calls consistency: %w", err)
	}

	if len(funcCalls) == 0 {
		return &ProcessingResult{
			Execution: UnprocessableRequestExecutions(),
		}, nil
	}

	exec, err := a.executeFunctionCalls(ctx, funcCalls, progress)
	if err != nil {
		return nil, fmt.Errorf("error executing functions: %w", err)
	}

	if a.config.AlterResult != nil {
		if err := a.config.AlterResult(&ProcessingResult{Execution: exec}); err != nil {
			return nil, fmt.Errorf("error on altering result: %w", err)
		}
	}

	return &ProcessingResult{
		Execution: exec,
	}, nil
}

func (a *RequestHandler) generateFunctionCalls(_ context.Context, message string, progress progress.Stream) ([]parser.PlannedFuncCall, error) {
	progress.Send("Generating system prompt...")
	systemPrompt, err := prompt.CreatePromptForFuncCalls(a.config.Tools.AvailableTools())
	if err != nil {
		return nil, fmt.Errorf("error generating system prompt: %w", err)
	}

	messages := []llm.Message{
		{"system", systemPrompt},
		{"user", message},
	}

	progress.Send("Generating schema for constrained generation...")
	jsonSchema, err := a.config.Tools.AvailableTools().ToJSONSchema()
	if err != nil {
		return nil, fmt.Errorf("failed to generate JSON schema: %w", err)
	}

	progress.Send("Thinking...")
	funcCallsCompletion, err := a.config.LLMClient.Complete(messages, string(jsonSchema))
	if err != nil {
		return nil, fmt.Errorf("error calling LLM: %w", err)
	}

	progress.Send("Synthesizing function calls...")
	return parser.ParseJsonFunctions([]byte(funcCallsCompletion))
}

func (a *RequestHandler) evaluateFuncCallsConsistency(message string, funcCalls []parser.PlannedFuncCall, progress progress.Stream) ([]parser.PlannedFuncCall, error) {
	if len(funcCalls) == 0 {
		return nil, nil
	}

	progress.Send("Evaluating function calls consistency...")
	result := make([]parser.PlannedFuncCall, 0, len(funcCalls))

	jsonSchema, err := json.Marshal(prompt.FuncCallsEvaluationResponseSchema)
	if err != nil {
		return nil, fmt.Errorf("error marshalling schema: %w", err)
	}

	at := a.config.Tools.AvailableTools()

	for _, function := range funcCalls {
		usedToolsName := function.CollectAllNestedFuncCalls()

		usedTools := make([]tools.FuncDefinition, 0, len(usedToolsName))
		for _, toolName := range usedToolsName {
			if tool, ok := at.FindTool(toolName); ok {
				usedTools = append(usedTools, *tool)
			} else {
				return nil, fmt.Errorf("tool %s not found", toolName)
			}
		}

		usedToolSet := tools.ToolSet{
			Functions:       usedTools,
			TypeDefinitions: at.TypeDefinitions,
		}

		if isConsistent, err := a.evaluateSingleFunctionCall(message, function, jsonSchema, &usedToolSet); err != nil {
			return nil, err
		} else if isConsistent {
			result = append(result, function)
		}
	}

	return result, nil
}

func (a *RequestHandler) evaluateSingleFunctionCall(message string, function parser.PlannedFuncCall, jsonSchema []byte, usedTools *tools.ToolSet) (bool, error) {
	data, err := json.MarshalIndent(function, "", "  ")
	if err != nil {
		return false, fmt.Errorf("error marshalling function: %w", err)
	}

	usedFunctionsJSON, err := usedTools.ToJSONDefinitions()
	if err != nil {
		return false, fmt.Errorf("error marshaling functions to JSON: %w", err)
	}

	userPrompt, err := prompt.CreatePromptForFuncCallsEvaluation(message, string(data), string(usedFunctionsJSON))
	if err != nil {
		return false, fmt.Errorf("error generating userPrompt for self-validation: %w", err)
	}

	body, err := a.config.LLMClient.Complete([]llm.Message{{"user", userPrompt}}, string(jsonSchema))
	if err != nil {
		return false, fmt.Errorf("error generating response for self-validation: %w", err)
	}

	var evaluation struct {
		Success bool `json:"success"`
	}

	if err := json.Unmarshal([]byte(body), &evaluation); err != nil {
		return false, fmt.Errorf("error unmarshaling JSON: %w", err)
	}

	a.config.Logger.Printf("Function %s -> %v", function.Name, evaluation.Success)
	return evaluation.Success, nil
}

func (a *RequestHandler) executeFunctionCalls(ctx context.Context, funcCalls []parser.PlannedFuncCall, progress progress.Stream) (*execution.Result, error) {
	if len(funcCalls) == 0 {
		return nil, fmt.Errorf("no function calls to execute")
	}
	progress.Send("Executing function calls...")
	return a.orchestrator.Execute(ctx, funcCalls, progress)
}
