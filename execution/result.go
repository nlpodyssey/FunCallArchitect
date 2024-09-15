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
	"fmt"
	"strings"
)

type Result struct {
	FuncCalls []*ExecutedFuncCall
}

func (e *Result) MainFuncResults() FuncResults {
	results := make([]FuncResult, len(e.FuncCalls))
	for i, f := range e.FuncCalls {
		results[i] = f.Result
	}
	return results
}

type ExecutedFuncCall struct {
	Name    string         `json:"name"`
	Purpose string         `json:"purpose"`
	Args    map[string]Arg `json:"args"`
	Result  FuncResult     `json:"-"`
}

type Arg interface{}

type ValueArg struct {
	Value interface{} `json:"value"`
}

func NewValueArg(v interface{}) Arg {
	return ValueArg{Value: v}
}

type FuncArg struct {
	Func *ExecutedFuncCall `json:"func_call"`
}

func NewFuncArg(c *ExecutedFuncCall) Arg {
	return FuncArg{Func: c}
}

func GetValue(arg Arg) (interface{}, bool) {
	if v, ok := arg.(ValueArg); ok {
		return v.Value, true
	}
	return nil, false
}

func GetFuncCall(arg Arg) (*ExecutedFuncCall, bool) {
	if f, ok := arg.(FuncArg); ok {
		return f.Func, true
	}
	return nil, false
}

// FormatFunc is a function that formats the execution result into a string.
// It returns the formatted string and any error encountered during formatting.
type FormatFunc func() (string, error)

type FuncResults []FuncResult

const DefaultSeparator = "\n\n---\n"

func (r FuncResults) Format(separator string) (string, error) {
	if separator == "" {
		separator = DefaultSeparator
	}

	var formatted []string
	for _, result := range r {
		if result.FormatFunc == nil {
			continue // skip silent functions
		}
		buf, err := result.FormatFunc()
		if err != nil {
			return "", fmt.Errorf("error formatting result: %v", err)
		}
		formatted = append(formatted, buf)
	}

	// Remove duplicates
	// TODO: Use a "Formatter" type to encapsulate this and the "separator" logic?
	seen := make(map[string]struct{})
	var unique []string
	for _, s := range formatted {
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			unique = append(unique, s)
		}
	}

	return strings.Join(unique, separator), nil
}

// FuncResult represents the outcome of a function execution.
// It contains the resulting data (if any), a flag indicating data presence,
// and a function to format the result.
type FuncResult struct {
	// Present indicates whether the execution resulted in data.
	// True if data was found/generated, false otherwise.
	Present bool

	// Value holds the actual data resulting from the execution.
	// It's of type interface{} to allow for flexibility in the type of data returned.
	// This field may be nil if Present is false.
	Value interface{}

	// FormatFunc is a function that knows how to format the execution result into a string.
	// It can handle both cases where data is present and where it's not.
	// The specific formatting logic is encapsulated within this function.
	FormatFunc FormatFunc

	// Metadata optionally provided by the function's implementation.
	Metadata any
}
