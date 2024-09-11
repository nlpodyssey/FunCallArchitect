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
	"fmt"

	"github.com/nlpodyssey/funcallarchitect/utils/orderedmap"
)

type funcDefsGenerator struct {
	Tools *ToolSet
}

// generateToolsDefinition generates a simplified version of the Tools definition for the prompt
func (t *funcDefsGenerator) generateToolsDefinition() (*orderedmap.Type, error) {
	simplifiedSchema := orderedmap.Map(
		orderedmap.Pair{Key: "functions", Value: []interface{}{}},
	)

	for _, function := range t.Tools.Functions {
		simplifiedFunction := orderedmap.Map(
			orderedmap.Pair{Key: "name", Value: function.Name},
			orderedmap.Pair{Key: "description", Value: function.Description},
			orderedmap.Pair{Key: "args", Value: t.getTypeInfo(function.Parameters, t.Tools.TypeDefinitions)},
		)

		functions := simplifiedSchema.Items[0].Value.([]interface{})
		simplifiedSchema.Items[0].Value = append(functions, simplifiedFunction)
	}

	return simplifiedSchema, nil
}

func (t *funcDefsGenerator) getTypeInfo(info TypeInfo, typeDefinitions map[string]TypeInfo) *orderedmap.Type {
	simplifiedType := orderedmap.Map(
		orderedmap.Pair{Key: "type", Value: t.getType(info, typeDefinitions)},
		orderedmap.Pair{Key: "description", Value: info.Description},
	)

	if info.Properties != nil {
		props := orderedmap.Map()
		for propName, propInfo := range info.Properties {
			props.Items = append(props.Items, orderedmap.Pair{Key: propName, Value: t.getTypeInfo(propInfo, typeDefinitions)})
		}
		simplifiedType.Items = append(simplifiedType.Items, orderedmap.Pair{Key: "properties", Value: props})
	}

	if info.Required != nil && len(info.Required) > 0 {
		simplifiedType.Items = append(simplifiedType.Items, orderedmap.Pair{Key: "required", Value: info.Required})
	}

	return simplifiedType
}

func (t *funcDefsGenerator) getType(typeInfo TypeInfo, typeDefinitions map[string]TypeInfo) string {
	if definition, exists := typeDefinitions[typeInfo.Type]; exists {
		if definition.Type == "object" {
			return typeInfo.Type // Return the name of the complex type
		}
		return t.getType(definition, typeDefinitions) // Recurse for aliases
	}

	if typeInfo.Type == "array" && typeInfo.Items != nil {
		itemType := t.getType(*typeInfo.Items, typeDefinitions)
		return fmt.Sprintf("array of %s", itemType)
	}

	return typeInfo.Type
}
