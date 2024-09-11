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

package orderedmap

import (
	"bytes"
	"encoding/json"
	"strings"
)

// Type represents an ordered map structure
type Type struct {
	Items []Pair
}

// Pair represents a key-value pair
type Pair struct {
	Key   string
	Value interface{}
}

// Map creates a new ordered map with initial items
func Map(pairs ...Pair) *Type {
	return &Type{Items: pairs}
}

// MarshalJSON implements the json.Marshaler interface
func (om *Type) MarshalJSON() ([]byte, error) {
	return om.MarshalJSONIndent(0)
}

func (om *Type) MarshalJSONIndent(indent int) ([]byte, error) {
	var buf bytes.Buffer
	indentStr := strings.Repeat("    ", indent)
	nextIndentStr := strings.Repeat("    ", indent+1)

	buf.WriteString("{\n")
	for i, pair := range om.Items {
		if i > 0 {
			buf.WriteString(",\n")
		}
		buf.WriteString(nextIndentStr)
		// Marshal key
		key, err := json.Marshal(pair.Key)
		if err != nil {
			return nil, err
		}
		buf.Write(key)
		buf.WriteString(": ")
		// Marshal value
		val, err := marshalJSONValue(pair.Value, indent+1)
		if err != nil {
			return nil, err
		}
		buf.Write(val)
	}
	buf.WriteString("\n")
	buf.WriteString(indentStr)
	buf.WriteString("}")
	return buf.Bytes(), nil
}

func marshalJSONValue(v interface{}, indent int) ([]byte, error) {
	switch val := v.(type) {
	case *Type:
		return val.MarshalJSONIndent(indent)
	case []interface{}:
		return marshalJSONArray(val, indent)
	default:
		return json.Marshal(v)
	}
}

func marshalJSONArray(arr []interface{}, indent int) ([]byte, error) {
	var buf bytes.Buffer
	indentStr := strings.Repeat("    ", indent)
	nextIndentStr := strings.Repeat("    ", indent+1)

	buf.WriteString("[\n")
	for i, item := range arr {
		if i > 0 {
			buf.WriteString(",\n")
		}
		buf.WriteString(nextIndentStr)
		val, err := marshalJSONValue(item, indent+1)
		if err != nil {
			return nil, err
		}
		buf.Write(val)
	}
	buf.WriteString("\n")
	buf.WriteString(indentStr)
	buf.WriteString("]")
	return buf.Bytes(), nil
}

func (om *Type) getOrderedMapProperty(key string) (*Type, bool) {
	for _, pair := range om.Items {
		if pair.Key == key {
			if value, ok := pair.Value.(*Type); ok {
				return value, true
			}
		}
	}
	return nil, false
}

func (om *Type) getStringProperty(key string) (string, bool) {
	for _, pair := range om.Items {
		if pair.Key == key {
			if value, ok := pair.Value.(string); ok {
				return value, true
			}
		}
	}
	return "", false
}

func (om *Type) getStringArrayProperty(key string) ([]string, bool) {
	for _, pair := range om.Items {
		if pair.Key == key {
			if value, ok := pair.Value.([]string); ok {
				return value, true
			}
		}
	}
	return nil, false
}
