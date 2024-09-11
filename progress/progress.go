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

package progress

// Stream sends real-time progress updates to a client.
// Implementations should ensure non-blocking Send operations.
type Stream interface {
	// Send transmits a progress message.
	// It should not block and should handle cases where sending might fail.
	Send(message string)
}

type NoOp struct{}

func (ne *NoOp) Send(_ string) {
	// Do nothing
}
