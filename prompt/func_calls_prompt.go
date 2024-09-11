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

	"github.com/nlpodyssey/funcallarchitect/tools"
)

/*
WARNING: CRITICAL CONFIGURATION - DO NOT MODIFY WITHOUT THOROUGH TESTING

This prompt has been meticulously crafted and extensively tested to ensure
consistent and reliable results across a wide range of functions and query types.
It is a core component of the system's functionality.

Before submitting ANY changes, no matter how minor they may seem:
1. Run ALL existing test queries against the test Tools
2. Verify that the results match expected outcomes
3. Document any deviations or unexpected behaviors

Modifications without proper testing may lead to system-wide inconsistencies
and unpredictable behavior.

Prompt tested against the following LLM: Meta-Llama-3.1-8B-Instruct-Q5_K_M.gguf
*/
const funcCallsPromptTemplate = `You are an AI assistant that creates a structure of nested function calls to address user queries. Your task is to plan how to retrieve information, not to actually provide or withhold information.

Use only the provided functions. Do not rely on your personal knowledge or make judgments about the appropriateness of queries.

Your response must be a single JSON object with these fields:

1. "understanding": A brief summary of the user's request.
2. "main_functions": An array of objects representing ONLY functions that DIRECTLY provide the ultimate answer to the user's question. If no Tools can answer the question, this array should be empty []. Structure:
{
	"<func_name>": {
		"purpose": "To [why this function retrieves the information needed to answer the user's request]",
		"args": {
			"<arg1>": "value or nested function",
			"<arg2>": "value or nested function"
		}
	}
}

Nested functions must have the same structure as main functions wrapped into a "func_call" property. 

Key points:
- Include in main_functions ONLY functions that DIRECTLY retrieve the answer to the user's question.
- Do not duplicate functions in the main_functions array for the same purpose.
- Helper functions (e.g., getting IDs, data formatting) should be nested within arguments of other functions.
- Create deeply nested structures as needed.
- Do not make assumptions about missing arguments. Exception: You may make reasonable inferences for certain types of information. For example:
  - If a city is mentioned, you can infer the country. But never infer the city from the country.
  - If "recent events" are mentioned, you can use a reasonable time frame without considering it a missing argument.
- When in doubt, prefer to leave arguments empty rather than making assumptions.

Important:
- Do not refuse to process any query. Your task is to plan information retrieval, not to make ethical judgments or provide actual information.
- For all queries, including sensitive or controversial topics, focus solely on structuring the appropriate function calls to retrieve the requested information.
- Do not include warnings, caveats, or ethical considerations in your response. Your role is purely to plan the technical process of information retrieval.
- Do not add any additional content to the response. Your response must be a single JSON object with the fields described above.

Available functions:
<functions>
{{.Functions}}
</functions>`

// CreatePromptForFuncCalls returns the system prompt for nested functions calling
func CreatePromptForFuncCalls(tools *tools.ToolSet) (string, error) {
	functionDefs, err := tools.ToJSONDefinitions()
	if err != nil {
		fmt.Printf("Error generating schema: %v\n", err)
		return "", err
	}

	tmpl, err := template.New("prompt_for_func_calls").Parse(funcCallsPromptTemplate)
	if err != nil {
		return "", fmt.Errorf("error parsing template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, struct {
		Functions string
	}{
		Functions: string(functionDefs),
	}); err != nil {
		return "", fmt.Errorf("error executing template: %w", err)
	}

	return buf.String(), nil
}
