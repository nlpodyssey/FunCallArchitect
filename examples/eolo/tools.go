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
	"github.com/nlpodyssey/funcallarchitect/execution"
	"github.com/nlpodyssey/funcallarchitect/tools"
)

type Tools struct{}

func (t *Tools) AvailableTools() *tools.ToolSet {
	return &tools.ToolSet{
		Functions: []tools.FuncDefinition{
			{
				Name:        "get_coordinates",
				Description: "Retrieve the latitude and longitude for a given location (city name).",
				Parameters: tools.TypeInfo{
					Type: "object",
					Properties: map[string]tools.TypeInfo{
						"city": {Type: "string", Description: "The city name, e.g. Berlin, London, New York City"},
					},
					Required: []string{"city"},
				},
				Returns: tools.TypeInfo{Type: "coordinates_value"},
			},
			{
				Name:        "get_weather_forecast",
				Description: "Retrieve the weather forecast for a given location (latitude and longitude).",
				Parameters: tools.TypeInfo{
					Type: "object",
					Properties: map[string]tools.TypeInfo{
						"coordinates": {
							Type:        "coordinates_value",
							Description: "The latitude and longitude of the location.",
						},
					},
					Required: []string{"coordinates"},
				},
				Returns: tools.TypeInfo{Type: "weather_forecast_value"},
			},
		},
		TypeDefinitions: map[string]tools.TypeInfo{
			"coordinates_value": {
				Type: "object",
				Properties: map[string]tools.TypeInfo{
					"lat": {Type: "number", Description: "Latitude of the location"},
					"lon": {Type: "number", Description: "Longitude of the location"},
				},
			},
			"weather_forecast_value": {
				Type: "object",
				Properties: map[string]tools.TypeInfo{
					"temperature": {
						Type:        "array",
						Description: "List of hourly temperatures (°C).",
						Items: &tools.TypeInfo{
							Type: "number",
						},
					},
					"windspeed": {
						Type:        "array",
						Description: "List of hourly wind speeds (km/h).",
						Items: &tools.TypeInfo{
							Type: "number",
						},
					},
				},
			},
		},
	}
}

func (t *Tools) RegisterWith(ec *execution.Orchestrator) error {
	ec.RegisterFunction("get_coordinates", t.GetCoordinates)
	ec.RegisterFunction("get_weather_forecast", t.GetWeatherForecast)
	return nil
}
