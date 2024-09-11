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

package agent

import (
	"context"

	"github.com/nlpodyssey/funcallarchitect/handler"
	"github.com/nlpodyssey/funcallarchitect/progress"
)

// Agent represents a high-level abstraction for processing user requests.
// It encapsulates a RequestHandler and provides a simplified interface for
// interpreting and executing user queries.
type Agent struct {
	requestHandler *handler.RequestHandler
}

type ProcessingResult struct {
	*handler.ProcessingResult
}

// NewAgent creates and initializes a new Agent with the given configuration.
func NewAgent(config handler.RequestHandlerConfig) (*Agent, error) {
	rh, err := handler.NewRequestHandler(config)
	if err != nil {
		return nil, err
	}
	return &Agent{requestHandler: rh}, nil
}

// Process interprets the user's message, executes the appropriate actions,
// and returns the processing result.
func (a *Agent) Process(ctx context.Context, message string, progress progress.Stream) (*ProcessingResult, error) {
	result, err := a.requestHandler.ProcessUserRequest(ctx, message, progress)
	if err != nil {
		return nil, err
	}
	return &ProcessingResult{ProcessingResult: result}, nil
}
