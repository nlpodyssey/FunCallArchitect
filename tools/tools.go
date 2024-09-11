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

package tools

import (
	"encoding/json"
	"fmt"
)

type ToolSet struct {
	Functions       []FuncDefinition    `json:"functions"`
	TypeDefinitions map[string]TypeInfo `json:"type_definitions"`
}

type FuncDefinition struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Parameters  TypeInfo `json:"parameters"`
	Returns     TypeInfo `json:"returns"`
}

type TypeInfo struct {
	Type        string              `json:"type"`
	Description string              `json:"description,omitempty"`
	Items       *TypeInfo           `json:"items,omitempty"`
	Properties  map[string]TypeInfo `json:"properties,omitempty"`
	Required    []string            `json:"required,omitempty"`
	Enum        []string            `json:"enum,omitempty"`
	Pattern     string              `json:"pattern,omitempty"`
}

func (t *ToolSet) ToJSONSchema() (json.RawMessage, error) {
	return (&toolsJSONSchemaGenerator{tools: t}).toJSONSchema()
}

func (t *ToolSet) ToJSONDefinitions() (json.RawMessage, error) {
	definitions, err := (&funcDefsGenerator{Tools: t}).generateToolsDefinition()
	if err != nil {
		fmt.Printf("error generating schema: %v\n", err)
		return nil, err
	}
	return definitions.MarshalJSON()
}

func (t *ToolSet) FindTool(name string) (*FuncDefinition, bool) {
	for _, function := range t.Functions {
		if function.Name == name {
			return &function, true
		}
	}
	return nil, false
}

func (t *ToolSet) ListTools() string {
	var result string
	for _, function := range t.Functions {
		result += fmt.Sprintf("%s: %s\n", function.Name, function.Description)
	}
	return result
}

func (t *ToolSet) isUsedAsArgumentType(typeName string) bool {
	for _, function := range t.Functions {
		if isTypeUsedInTypeInfo(typeName, function.Parameters) {
			return true
		}
	}
	return false
}
