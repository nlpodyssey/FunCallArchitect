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

package main

import (
	"bufio"
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
)

//go:embed template.html
var templateFS embed.FS

//go:embed favicon.ico
var favicon []byte

type Config struct {
	CompanyNamePrefix  string `yaml:"company_name_prefix"`
	CompanyNameSuffix  string `yaml:"company_name_suffix"`
	CompanySuffixColor string `yaml:"company_suffix_color"`
	ProductName        string `yaml:"product_name"`
	EnvironmentName    string `yaml:"environment_name"`
	ProductAccentColor string `yaml:"product_accent_color"`
	FaviconPath        string `yaml:"favicon_path"`
	InitialQuestion    string `yaml:"initial_question"`
}

func loadConfig(filename string) (*Config, error) {
	config := &Config{}

	file, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %v", err)
	}

	err = yaml.Unmarshal(file, config)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %v", err)
	}

	return config, nil
}

func main() {
	config, err := loadConfig("config.yaml")
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	htmlTemplate, err := templateFS.ReadFile("template.html")
	if err != nil {
		fmt.Println("Error reading embedded template file:", err)
		return
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.New("index").Parse(string(htmlTemplate))
		if err != nil {
			http.Error(w, "Error parsing template", http.StatusInternalServerError)
			return
		}
		err = tmpl.Execute(w, config)
		if err != nil {
			return
		}
	})

	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/x-icon")
		w.Write(favicon)
	})

	http.HandleFunc("/process", handleProcess)

	fmt.Println("Server is running on http://localhost:8080")
	err = http.ListenAndServe(":8080", nil) // TODO: Update port
	if err != nil {
		log.Fatal(err)
	}
}

func handleProcess(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	message, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	url := "http://localhost:8081/process" // TODO: Update URL
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(message))
	if err != nil {
		http.Error(w, "Error creating request", http.StatusInternalServerError)
		return
	}

	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Accept", "text/event-stream")

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Error sending request", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			_, err := fmt.Fprintf(w, "data: Error reading response: %v\n\n", err)
			if err != nil {
				return
			}
			return
		}

		_, err = fmt.Fprint(w, line)
		if err != nil {
			return
		}
		w.(http.Flusher).Flush()

		if strings.HasPrefix(strings.TrimSpace(line), "data: err:") {
			break
		}
	}
}
