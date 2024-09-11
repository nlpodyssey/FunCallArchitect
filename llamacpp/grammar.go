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

package llamacpp

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"
	"sync"
)

//go:embed json_schema_to_grammar.py
var pythonScript string

// GrammarCache represents a thread-safe cache for grammars
type GrammarCache struct {
	sync.RWMutex
	m map[string]string
}

// Get retrieves a grammar from the cache if it exists
func (gc *GrammarCache) Get(hash string) (string, bool) {
	gc.RLock()
	defer gc.RUnlock()
	grammar, found := gc.m[hash]
	return grammar, found
}

// Set adds a grammar to the cache
func (gc *GrammarCache) Set(hash, grammar string) {
	gc.Lock()
	defer gc.Unlock()
	gc.m[hash] = grammar
}

// Global cache instance
var grammarCache = &GrammarCache{
	m: make(map[string]string),
}

// jsonSchemaToGrammar generates a BNF grammar from a JSON schema
func jsonSchemaToGrammar(jsonSchema string) (string, error) {
	hash, err := calculateFingerprint(jsonSchema)
	if err != nil {
		return "", fmt.Errorf("failed to calculate fingerprint: %w", err)
	}

	if grammar, found := grammarCache.Get(hash); found {
		log.Println("Grammar found in cache")
		return grammar, nil
	}

	log.Println("Generating grammar from JSON schema")
	cmd := exec.Command("python3", "-c", pythonScript, "-")

	grammar, err := runPythonCommand(cmd, jsonSchema)
	if err != nil {
		return "", fmt.Errorf("failed to run Python command: %w", err)
	}

	grammarCache.Set(hash, grammar)

	return grammar, nil
}

// runPythonCommand executes a Python command with the given input and returns the output
func runPythonCommand(cmd *exec.Cmd, input string) (string, error) {
	var stdout bytes.Buffer
	cmd.Stdin = bytes.NewBufferString(input)
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", err
	}

	return stdout.String(), nil
}

func calculateFingerprint(str string) (string, error) {
	var data interface{}
	if err := json.Unmarshal([]byte(str), &data); err != nil {
		return "", fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	// Ensure consistent string representation
	result, err := json.Marshal(orderJSON(data))
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON schema: %w", err)
	}

	hash := sha256.Sum256(result)
	return hex.EncodeToString(hash[:]), nil
}

// orderJSON recursively orders elements in maps and slices
func orderJSON(data interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		newMap := make(map[string]interface{})
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			newMap[k] = orderJSON(v[k])
		}
		return newMap
	case []interface{}:
		for i, item := range v {
			v[i] = orderJSON(item)
		}
		return v
	default:
		return v
	}
}
