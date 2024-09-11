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

package prompt

import (
	"bytes"
	"fmt"
	"text/template"
)

/*
WARNING: CRITICAL CONFIGURATION - DO NOT MODIFY WITHOUT THOROUGH TESTING

This prompt is a core component of the system's functionality.

Before submitting ANY changes, no matter how minor they may seem:
1. Run ALL existing test queries against the test Tools
2. Verify that the results match expected outcomes
3. Document any deviations or unexpected behaviors

Modifications without proper testing may lead to system-wide inconsistencies
and unpredictable behavior.

Prompt tested against the following LLM: Meta-Llama-3.1-8B-Instruct-Q5_K_M.gguf
*/
const funcCallsEvaluationPromptTemplate = `You are a function call validator. Your task is to evaluate the provided function calls based on the user's initial request. You will be provided with function definitions, function calls to be validated, and the initial user request.

Evaluate whether the function calls correctly fulfill the user's request based on the criteria provided below. Focus on correctness and appropriateness of the function calls, not on providing or withholding information. Do not make judgments about the appropriateness of queries.

Evaluation Criteria:
1. The main function must directly address at least one aspect of the user request (partial fulfillment of compound requests is acceptable for the main function).
2. All function arguments (main and nested) must have values that are:
   a) Explicitly mentioned in the user request, OR
   b) Default values provided in the function definitions, OR
   c) Obtainable via nested functions.
   b) Inferable from the user request without making assumptions. Exception: You may make reasonable inferences for certain types of information. For example:
      - If a city is mentioned, you can accept inferred the country. But never infer the city from the country.

Key Points:
- A compound request is a user request that contains multiple distinct tasks, questions, or requirements. For example, "Tell me the weather and send an email" is a compound request.
- Successfully completing any single part of the request would satisfy the assessment requirements.
- Nested functions should support the main function but aren't evaluated for direct request fulfillment.
- Intermediate steps or preparatory functions do not qualify as main functions.
- Do not accept arguments with assumed or estimated values not based on the user request.
- Reject function calls that introduce information not present in or directly inferable from the user request.

Response Format:
{
  "success": boolean
}

Set "success" to true if the main function call addresses at least one aspect of the user request and all function arguments meet the specified criteria. Otherwise, set it to false.

Important:
- Do not refuse to process any query. Your task is to validate function calls, not to make ethical judgments or provide actual information.
- For all queries, including sensitive or controversial topics, focus solely on validating the appropriate function calls to retrieve the requested information.
- When processing user queries, it's crucial to look beyond the literal words and understand the underlying intent or broader context of the question.
- Do not include warnings, caveats, or ethical considerations in your response. Your role is purely to plan the technical process of validating the function calls.
- Do not add any explanation or additional content to the response. Your response must be a single JSON object with the fields described above.

---
Function Definitions (for reference):
{{.FuncDefinitions}}

Function Calls to Validate:
{{.PlannedFuncCalls}}

Initial User Request: 
{{.UserRequest}}`

var FuncCallsEvaluationResponseSchema = map[string]any{
	"$schema": "http://json-schema.org/draft-07/schema#",
	"type":    "object",
	"properties": map[string]any{
		"success": map[string]any{
			"type": "boolean",
		},
	},
	"required":             []string{"success"},
	"additionalProperties": false,
}

// CreatePromptForFuncCallsEvaluation generates a prompt for a second-pass function call validation
func CreatePromptForFuncCallsEvaluation(userRequest, plannedFuncCalls, funcDefinitions string) (string, error) {
	tmpl, err := template.New("prompt_for_func_calls_evaluation").Parse(funcCallsEvaluationPromptTemplate)
	if err != nil {
		return "", fmt.Errorf("error parsing template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, struct {
		UserRequest      string
		PlannedFuncCalls string
		FuncDefinitions  string
	}{
		UserRequest:      userRequest,
		PlannedFuncCalls: plannedFuncCalls,
		FuncDefinitions:  funcDefinitions,
	}); err != nil {
		return "", fmt.Errorf("error executing template: %w", err)
	}

	return buf.String(), nil
}
