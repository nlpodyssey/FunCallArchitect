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

package execution

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/nlpodyssey/funcallarchitect/parser"
	"github.com/nlpodyssey/funcallarchitect/progress"
	"github.com/nlpodyssey/funcallarchitect/tools"
	"golang.org/x/sync/errgroup"
)

// FuncExecutor represents a function that can be executed, performs some operation and returns a FuncResult.
// It takes a context for cancellation, a map of arguments, and a channel for logging.
// It returns a FuncResult and an error. The error is used for execution failures,
// while FuncResult.Present indicates whether data was found/generated.
type FuncExecutor func(ctx context.Context, args map[string]interface{}, progress progress.Stream) (FuncResult, error)

// Orchestrator holds the context for function execution, including memoization
type Orchestrator struct {
	Functions map[string]FuncExecutor
	Memo      map[string]FuncResult
	MemoLock  sync.RWMutex
	Logger    *log.Logger
	Timeout   time.Duration

	EnableConcurrentExec bool
	ToolSet              *tools.ToolSet
}

// Error represents an error that occurred during function execution
type Error struct {
	FuncName string
	ArgName  string
	Err      error
}

func (e *Error) Error() string {
	if e.ArgName != "" {
		return fmt.Sprintf("error in function '%s' for argument '%s': %v", e.FuncName, e.ArgName, e.Err)
	}
	return fmt.Sprintf("error in function '%s': %v", e.FuncName, e.Err)
}

type FormattableError struct {
	FormatFunc FormatFunc
}

func NewFormattableError(formatFunc FormatFunc) *FormattableError {
	return &FormattableError{FormatFunc: formatFunc}
}

func (e *FormattableError) Error() string {
	result, err := e.FormatFunc()
	if err != nil {
		return fmt.Sprintf("FormattableError<FormatFunc error: %v>", err)
	}
	return result
}

func IsFormattableError(err error) bool {
	return errors.Is(err, &FormattableError{})
}

func AsFormattableError(err error) (*FormattableError, bool) {
	var f *FormattableError
	ok := errors.As(err, &f)
	return f, ok
}

// NewOrchestrator creates a new Orchestrator
func NewOrchestrator(logger *log.Logger, timeout time.Duration, enableConcurrentExec bool, toolSet *tools.ToolSet) *Orchestrator {
	return &Orchestrator{
		Functions:            make(map[string]FuncExecutor),
		Memo:                 make(map[string]FuncResult),
		Logger:               logger,
		Timeout:              timeout,
		EnableConcurrentExec: enableConcurrentExec,
		ToolSet:              toolSet,
	}
}

// RegisterFunction registers a function executor with the context
func (o *Orchestrator) RegisterFunction(name string, executor FuncExecutor) {
	o.Functions[name] = executor
}

// Execute executes a slice of PlannedFuncCall and returns the results
func (o *Orchestrator) Execute(ctx context.Context, functions []parser.PlannedFuncCall, progress progress.Stream) (*Result, error) {
	if o.EnableConcurrentExec {
		return o.executeConcurrent(ctx, functions, progress)
	}
	return o.executeSeq(ctx, functions, progress)
}

func (o *Orchestrator) executeSeq(ctx context.Context, functions []parser.PlannedFuncCall, progress progress.Stream) (*Result, error) {
	functionsExecution := make([]*ExecutedFuncCall, len(functions))

	for i, function := range functions {
		o.Logger.Printf("Executing function: %s", function.Name)
		funcExe, err := o.executeFunc(ctx, function, progress)
		if err != nil {
			return nil, &Error{FuncName: function.Name, Err: err}
		}
		functionsExecution[i] = funcExe
		o.Logger.Printf("Function %s executed successfully", function.Name)
	}

	exe := &Result{FuncCalls: functionsExecution}
	return exe, nil
}

// executeConcurrent executes a slice of PlannedFuncCall concurrently using errgroup and returns the results
func (o *Orchestrator) executeConcurrent(ctx context.Context, functions []parser.PlannedFuncCall, progress progress.Stream) (*Result, error) {
	group, ctx := errgroup.WithContext(ctx)
	functionsExecution := make([]*ExecutedFuncCall, len(functions))

	for i, function := range functions {
		i, function := i, function
		group.Go(func() error {
			o.Logger.Printf("Executing function: %s", function.Name)
			funcExe, err := o.executeFunc(ctx, function, progress)
			if err != nil {
				return &Error{FuncName: function.Name, Err: err}
			}
			functionsExecution[i] = funcExe
			o.Logger.Printf("Function %s executed successfully", function.Name)
			return nil
		})
	}

	// Wait for all functions to complete or for an error to occur
	if err := group.Wait(); err != nil {
		return nil, err
	}

	exe := &Result{FuncCalls: functionsExecution}
	return exe, nil
}

// executeFunc executes a single PlannedFunctionCall
func (o *Orchestrator) executeFunc(ctx context.Context, function parser.PlannedFuncCall, progress progress.Stream) (*ExecutedFuncCall, error) {
	executor, ok := o.Functions[function.Name]
	if !ok {
		return nil, &Error{FuncName: function.Name, Err: fmt.Errorf("unknown function")}
	}

	// Process arguments, executing nested functions if necessary
	argsExecution, err := o.processArgs(ctx, function, progress)
	if err != nil {
		return nil, err
	}

	// Check for required arguments
	if err = o.checkRequiredArgs(function, argsExecution); err != nil {
		return handleMissingRequiredArgsError(err, function, argsExecution)
	}

	processedArgs := createProcessedArgs(argsExecution)

	// Generate a fingerprint for memoization
	fingerprint := generateFingerprint(function.Name, processedArgs)

	// Check memoization cache
	o.MemoLock.RLock()
	if result, ok := o.Memo[fingerprint]; ok {
		o.MemoLock.RUnlock()
		o.Logger.Printf("Using memoized result for function: %s", function.Name)
		return &ExecutedFuncCall{
			Name:    function.Name,
			Purpose: function.Purpose,
			Args:    argsExecution,
			Result:  result,
		}, nil
	}
	o.MemoLock.RUnlock()

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(ctx, o.Timeout)
	defer cancel()

	// Execute the function with timeout
	resultChan := make(chan FuncResult, 1)
	errChan := make(chan error, 1)

	go func() {
		result, err := executor(ctx, processedArgs, progress)
		if err != nil {
			errChan <- &Error{FuncName: function.Name, Err: err}
		} else {
			resultChan <- result
		}
	}()

	select {
	case result := <-resultChan:
		// Store the result in memoization cache
		o.MemoLock.Lock()
		o.Memo[fingerprint] = result
		o.MemoLock.Unlock()
		o.Logger.Printf("Function %s executed and result memoized", function.Name)

		return &ExecutedFuncCall{
			Name:    function.Name,
			Purpose: function.Purpose,
			Args:    argsExecution,
			Result:  result,
		}, nil
	case err := <-errChan:
		o.Logger.Printf("Error executing function %s: %v", function.Name, err)
		return nil, err
	case <-ctx.Done():
		o.Logger.Printf("Function %s timed out", function.Name)
		return nil, &Error{FuncName: function.Name, Err: fmt.Errorf("function execution timed out")}
	}
}

func handleMissingRequiredArgsError(err error, function parser.PlannedFuncCall, argsExecution map[string]Arg) (*ExecutedFuncCall, error) {
	fe, ok := AsFormattableError(err)
	if !ok {
		return nil, err
	}
	return &ExecutedFuncCall{
		Name:    function.Name,
		Purpose: function.Purpose,
		Args:    argsExecution,
		Result: FuncResult{
			Present:    false,
			Value:      nil,
			FormatFunc: fe.FormatFunc,
			Metadata:   nil,
		},
	}, nil
}

// processArgs processes the arguments, executing nested functions if necessary
func (o *Orchestrator) processArgs(ctx context.Context, function parser.PlannedFuncCall, progress progress.Stream) (map[string]Arg, error) {
	args := make(map[string]Arg)

	for key, value := range function.Args {
		switch v := value.(type) {
		case *parser.PlannedFuncCall:
			o.Logger.Printf("Processing nested function for argument '%s' in function '%s'", key, function.Name)
			funcExe, err := o.executeFunc(ctx, *v, progress)
			if err != nil {
				return nil, &Error{FuncName: function.Name, ArgName: key, Err: err}
			}
			args[key] = NewFuncArg(funcExe)
		default:
			args[key] = NewValueArg(v)
		}
	}
	return args, nil
}

func createProcessedArgs(argsExecution map[string]Arg) map[string]any {
	processedArgs := make(map[string]any)

	for key, arg := range argsExecution {
		switch v := arg.(type) {
		case FuncArg:
			if v.Func.Result.Present {
				processedArgs[key] = v.Func.Result.Value
			}
		case ValueArg:
			processedArgs[key] = v.Value
		}
	}
	return processedArgs
}

// checkRequiredArgs checks if all required arguments are present
func (o *Orchestrator) checkRequiredArgs(function parser.PlannedFuncCall, args map[string]Arg) error {
	functionSchema, ok := o.ToolSet.FindTool(function.Name)
	if !ok {
		return fmt.Errorf("function schema not found for %s", function.Name)
	}

	for _, paramName := range functionSchema.Parameters.Required {
		if err := o.checkRequiredArg(paramName, args); err != nil {
			return &Error{
				FuncName: function.Name,
				ArgName:  paramName,
				Err:      err,
			}
		}
	}

	return nil
}

func (o *Orchestrator) checkRequiredArg(paramName string, args map[string]Arg) error {
	arg, ok := args[paramName]
	if !ok {
		return fmt.Errorf("missing argument for required parameter %s", paramName)
	}

	funcCall, ok := GetFuncCall(arg)
	if !ok || funcCall.Result.Present {
		return nil
	}

	if ff := funcCall.Result.FormatFunc; ff != nil {
		return NewFormattableError(ff)
	}

	return fmt.Errorf("missing argument for required parameter %s: func call result is blank and has no FormatFunc", paramName)
}

// generateFingerprint creates a unique fingerprint for a function call
func generateFingerprint(functionName string, args map[string]interface{}) string {
	data, _ := json.Marshal(struct {
		Name string
		Args map[string]interface{}
	}{
		Name: functionName,
		Args: args,
	})
	return fmt.Sprintf("%x", sha256.Sum256(data))
}
