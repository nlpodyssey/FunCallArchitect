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

package parser

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
)

// ErrInvalidJSON is returned when the JSON structure is invalid
var ErrInvalidJSON = errors.New("invalid JSON structure")

// PlannedFuncCall represents a parsed function with its name, purpose, and arguments
type PlannedFuncCall struct {
	Name    string                 `json:"name"`
	Purpose string                 `json:"purpose"`
	Args    map[string]interface{} `json:"args"`
}

func (t *PlannedFuncCall) CollectAllNestedFuncCalls() []string {
	var nestedFuncCalls []string
	nestedFuncCalls = append(nestedFuncCalls, t.Name)
	for _, arg := range t.Args {
		switch v := arg.(type) {
		case *PlannedFuncCall:
			nestedFuncCalls = append(nestedFuncCalls, v.CollectAllNestedFuncCalls()...)
		}
	}
	return nestedFuncCalls
}

// ParseJsonFunctions parses the input JSON data and returns a slice of PlannedFunctionCall
func ParseJsonFunctions(jsonData []byte) ([]PlannedFuncCall, error) {
	var data struct {
		Understanding string        `json:"understanding"`
		MainFunctions []interface{} `json:"main_functions"`
	}

	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("error unmarshalling JSON: %w", err)
	}

	if len(data.MainFunctions) == 0 {
		log.Println("main_functions is empty")
		return nil, nil
	}

	var parsedFunctions []PlannedFuncCall

	for _, funcInterface := range data.MainFunctions {
		funcMap, ok := funcInterface.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("%w: function not a map", ErrInvalidJSON)
		}

		if len(funcMap) != 1 {
			return nil, fmt.Errorf("%w: function map should contain exactly one key-value pair", ErrInvalidJSON)
		}

		for funcName, funcDetails := range funcMap {
			parsedFunc, err := parseFuncDetails(funcName, funcDetails)
			if err != nil {
				return nil, err
			}
			parsedFunctions = append(parsedFunctions, parsedFunc)
		}
	}

	return parsedFunctions, nil
}

// parseFuncDetails parses the details of a single function
func parseFuncDetails(funcName string, funcDetails interface{}) (PlannedFuncCall, error) {
	detailsMap, ok := funcDetails.(map[string]interface{})
	if !ok {
		return PlannedFuncCall{}, fmt.Errorf("%w: function details not a map", ErrInvalidJSON)
	}

	purpose, ok := detailsMap["purpose"].(string)
	if !ok {
		return PlannedFuncCall{}, fmt.Errorf("%w: purpose not found or not a string", ErrInvalidJSON)
	}

	args, ok := detailsMap["args"].(map[string]interface{})
	if !ok {
		return PlannedFuncCall{}, fmt.Errorf("%w: args not found or not a map", ErrInvalidJSON)
	}

	parsedArgs, err := parseArgs(args)
	if err != nil {
		return PlannedFuncCall{}, err
	}

	return PlannedFuncCall{
		Name:    funcName,
		Purpose: purpose,
		Args:    parsedArgs,
	}, nil
}

// parseArgs parses the arguments of a function, handling nested function calls
func parseArgs(args map[string]interface{}) (map[string]interface{}, error) {
	parsedArgs := make(map[string]interface{})

	for key, value := range args {
		switch v := value.(type) {
		case map[string]interface{}:
			if funcCall, ok := v["func_call"].(map[string]interface{}); ok {
				nestedFunc, err := parseNestedFunc(funcCall)
				if err != nil {
					return nil, fmt.Errorf("error parsing nested function for arg '%s': %w", key, err)
				}
				parsedArgs[key] = nestedFunc
			} else {
				parsedArgs[key] = v
			}
		default:
			if strValue, ok := v.(string); ok {
				if strValue != "" {
					parsedArgs[key] = v
				}
			} else {
				parsedArgs[key] = v
			}
		}
	}

	return parsedArgs, nil
}

// parseNestedFunc parses a nested function call
func parseNestedFunc(funcCall map[string]interface{}) (*PlannedFuncCall, error) {
	if len(funcCall) != 1 {
		return nil, fmt.Errorf("%w: nested function call should contain exactly one key-value pair", ErrInvalidJSON)
	}

	for funcName, funcDetails := range funcCall {
		parsedFunc, err := parseFuncDetails(funcName, funcDetails)
		if err != nil {
			return nil, fmt.Errorf("error parsing nested function '%s': %w", funcName, err)
		}
		return &parsedFunc, nil
	}

	return nil, fmt.Errorf("%w: no valid nested function found", ErrInvalidJSON)
}
