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
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"
)

type toolsJSONSchemaGenerator struct {
	tools *ToolSet
}

func (t *toolsJSONSchemaGenerator) toJSONSchema() (json.RawMessage, error) {
	schemaTemplate := `{
        "$schema": "http://json-schema.org/draft-07/schema#",
        "type": "object",
        "properties": {
            "understanding": {
                "type": "string"
            },
            "main_functions": {
                "type": "array",
                "items": {
                    "$ref": "#/$defs/func_call"
                }
            }
        },
        "required": ["understanding", "main_functions"],
        "additionalProperties": false,
        "$defs": {
            "func_call": {
                "oneOf": [{{range $index, $element := .FuncCallOneOf}}{{if $index}},{{end}}{"$ref": "#/$defs/{{$element}}"}{{end}}]
            },
            {{range $index, $element := .Defs}}{{if $index}},{{end}}{{$element}}{{end}}
        }
    }`

	tmpl, err := template.New("schema").Parse(schemaTemplate)
	if err != nil {
		return nil, fmt.Errorf("error parsing schema template: %w", err)
	}

	var funcCallOneOf []string
	var defs []string

	for _, function := range t.tools.Functions {
		funcCallOneOf = append(funcCallOneOf, function.Name)

		funcDef, err := t.generateFunctionDefinition(function)
		if err != nil {
			return nil, fmt.Errorf("error generating function definition for %s: %w", function.Name, err)
		}
		defs = append(defs, fmt.Sprintf(`"%s": %s`, function.Name, string(funcDef)))
	}

	for typeName, typeInfo := range t.tools.TypeDefinitions {
		typeDef, err := t.generateTypeDefinition(typeName, typeInfo)
		if err != nil {
			return nil, fmt.Errorf("error generating type definition for %s: %w", typeName, err)
		}
		defs = append(defs, fmt.Sprintf(`"%s": %s`, typeName, string(typeDef)))
	}

	for defName, defValue := range t.generateFuncCallReturningDefinitions() {
		defs = append(defs, fmt.Sprintf(`"%s": %s`, defName, string(defValue)))
	}

	var fullSchema bytes.Buffer
	err = tmpl.Execute(&fullSchema, map[string]interface{}{
		"FuncCallOneOf": funcCallOneOf,
		"Defs":          defs,
	})
	if err != nil {
		return nil, fmt.Errorf("error executing schema template: %w", err)
	}

	// Compact the generated schema
	var compactSchema bytes.Buffer
	err = json.Compact(&compactSchema, fullSchema.Bytes())
	if err != nil {
		return nil, fmt.Errorf("error compacting schema: %w", err)
	}

	return compactSchema.Bytes(), nil
}

func (t *toolsJSONSchemaGenerator) generateFunctionDefinition(function FuncDefinition) (json.RawMessage, error) {
	schemaTemplate := `{
        "type": "object",
        "additionalProperties": false,
        "required": ["{{.Name}}"],
        "properties": {
            "{{.Name}}": {
                "type": "object",
                "description": "{{.Description}}",
                "additionalProperties": false,
                "required": ["purpose", "args"],
                "properties": {
                    "purpose": {
                        "type": "string"
                    },
                    "args": {{.Args}}
                }
            }
        }
    }`

	tmpl, err := template.New("funcDef").Parse(schemaTemplate)
	if err != nil {
		return nil, fmt.Errorf("error parsing function definition template: %w", err)
	}

	args, err := t.transformTypeInfo(function.Parameters, t.tools.TypeDefinitions)
	if err != nil {
		return nil, fmt.Errorf("error transforming parameters: %w", err)
	}

	var fullSchema bytes.Buffer
	err = tmpl.Execute(&fullSchema, map[string]interface{}{
		"Name":        function.Name,
		"Description": function.Description,
		"Args":        string(args),
	})
	if err != nil {
		return nil, fmt.Errorf("error executing function definition template: %w", err)
	}

	return fullSchema.Bytes(), nil
}

func (t *toolsJSONSchemaGenerator) transformTypeInfo(info TypeInfo, typeDefinitions map[string]TypeInfo) (json.RawMessage, error) {
	// Check if this is a custom type
	if _, exists := typeDefinitions[info.Type]; exists {
		return json.RawMessage(fmt.Sprintf(`{"$ref": "#/$defs/%s"}`, info.Type)), nil
	}

	baseTemplate := `{
		"type": "{{.Type}}"
		{{- if .Description -}}
		,"description": {{.Description | printf "%q"}}
		{{- end -}}
		{{- if .Enum -}}
		,"enum": {{.Enum}}
		{{- end -}}
		{{- if .Pattern -}}
		,"pattern": {{.Pattern | printf "%q"}}
		{{- end -}}
		{{- if .Items -}}
		,"items": {{.Items}}
		{{- end -}}
		{{- if .Properties -}}
		,"properties": { {{.Properties}} }
		,"additionalProperties": false
		{{- end -}}
		{{- if .Required -}}
		,"required": {{.Required}}
		{{- end -}}
	}`

	tmpl, err := template.New("typeInfo").Parse(baseTemplate)
	if err != nil {
		return nil, fmt.Errorf("error parsing type info template: %w", err)
	}

	var additionalProps struct {
		Type        string
		Description string
		Enum        string
		Pattern     string
		Items       string
		Properties  string
		Required    string
	}

	additionalProps.Type = info.Type
	additionalProps.Description = info.Description

	if info.Enum != nil && len(info.Enum) > 0 {
		enumJSON, err := json.Marshal(info.Enum)
		if err != nil {
			return nil, fmt.Errorf("error marshaling enum: %w", err)
		}
		additionalProps.Enum = string(enumJSON)
	}

	additionalProps.Pattern = info.Pattern

	if info.Items != nil {
		items, err := t.transformTypeInfo(*info.Items, typeDefinitions)
		if err != nil {
			return nil, fmt.Errorf("error transforming items: %w", err)
		}
		additionalProps.Items = string(items)
	}

	if info.Properties != nil {
		var propertyStrings []string
		for name, propInfo := range info.Properties {
			propDef, err := t.transformTypeInfo(propInfo, typeDefinitions)
			if err != nil {
				return nil, fmt.Errorf("error transforming property %s: %w", name, err)
			}
			propertyStrings = append(propertyStrings, fmt.Sprintf("%q: %s", name, string(propDef)))
		}
		additionalProps.Properties = strings.Join(propertyStrings, ",")

		if info.Required != nil && len(info.Required) > 0 {
			requiredJSON, err := json.Marshal(info.Required)
			if err != nil {
				return nil, fmt.Errorf("error marshaling required fields: %w", err)
			}
			additionalProps.Required = string(requiredJSON)
		}
	}

	var result bytes.Buffer
	if err := tmpl.Execute(&result, additionalProps); err != nil {
		return nil, fmt.Errorf("error executing type info template: %w", err)
	}

	// Compact the JSON
	var compactResult bytes.Buffer
	if err := json.Compact(&compactResult, result.Bytes()); err != nil {
		return nil, fmt.Errorf("error compacting JSON: %w", err)
	}

	return compactResult.Bytes(), nil
}

func (t *toolsJSONSchemaGenerator) generateFuncCallReturningDefinitions() map[string]json.RawMessage {
	definitions := make(map[string]json.RawMessage)
	typeToFunctions := make(map[string][]string)

	for _, function := range t.tools.Functions {
		returnType := function.Returns.Type
		if t.tools.isUsedAsArgumentType(returnType) {
			typeToFunctions[returnType] = append(typeToFunctions[returnType], function.Name)
		}
	}

	definitionTemplate := `{
        "type": "object",
        "required": ["func_call"],
        "additionalProperties": false,
        "properties": {
            "func_call": {
                "oneOf": [{{range $index, $element := .OneOf}}{{if $index}},{{end}}{"$ref": "#/$defs/{{$element}}"}{{end}}]
            }
        }
    }`

	tmpl, err := template.New("funcCallReturning").Parse(definitionTemplate)
	if err != nil {
		panic(fmt.Sprintf("error parsing func_call_returning template: %v", err))
	}

	for typeName, functionNames := range typeToFunctions {
		defName := fmt.Sprintf("func_call_returning_%s", typeName)

		var definitionBuffer bytes.Buffer
		err = tmpl.Execute(&definitionBuffer, map[string]interface{}{
			"OneOf": functionNames,
		})
		if err != nil {
			panic(fmt.Sprintf("error executing func_call_returning template: %v", err))
		}

		definitions[defName] = definitionBuffer.Bytes()
	}

	return definitions
}

func (t *toolsJSONSchemaGenerator) generateTypeDefinition(typeName string, typeInfo TypeInfo) (json.RawMessage, error) {
	baseDef, err := t.transformTypeInfo(typeInfo, t.tools.TypeDefinitions)
	if err != nil {
		return nil, fmt.Errorf("error transforming type info for %s: %w", typeName, err)
	}

	// Check if this type is used as an argument type (either directly or as an array item)
	if t.tools.isUsedAsArgumentType(typeName) {
		typeDefTemplate := `{"oneOf": [%s, {"$ref": "#/$defs/func_call_returning_%s"}]}`
		return json.RawMessage(fmt.Sprintf(typeDefTemplate, string(baseDef), typeName)), nil
	}

	return baseDef, nil
}

func isTypeUsedInTypeInfo(typeName string, info TypeInfo) bool {
	if info.Type == typeName {
		return true
	}
	if info.Type == "array" && info.Items != nil && info.Items.Type == typeName {
		return true
	}
	if info.Properties != nil {
		for _, propInfo := range info.Properties {
			if isTypeUsedInTypeInfo(typeName, propInfo) {
				return true
			}
		}
	}
	return false
}
