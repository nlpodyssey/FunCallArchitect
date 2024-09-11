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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"

	"github.com/nlpodyssey/funcallarchitect/execution"
	"github.com/nlpodyssey/funcallarchitect/progress"
)

type GetWeatherForecastResponse struct {
	Temperature2M []float64 `json:"temperature_2m"`
	WindSpeed10M  []float64 `json:"windspeed_10m"`
}

// WeatherData represents the structure of the weather data returned from the Open-Meteo API
type WeatherData struct {
	Hourly struct {
		Temperature2M []float64 `json:"temperature_2m"`
		WindSpeed10M  []float64 `json:"windspeed_10m"`
	} `json:"hourly"`
}

func (t *Tools) GetWeatherForecast(_ context.Context, args map[string]interface{}, progress progress.Stream) (execution.FuncResult, error) {
	coordinates, err := argsToGetWeatherForecastRequest(args)
	if err != nil {
		return execution.FuncResult{}, fmt.Errorf("coordinates argument is required")
	}

	latitude, longitude := coordinates.Lat, coordinates.Lon

	progress.Send(fmt.Sprintf("Retrieving weather forecast for %f, %f...", latitude, longitude))

	url := fmt.Sprintf("https://api.open-meteo.com/v1/forecast?latitude=%f&longitude=%f&hourly=temperature_2m,windspeed_10m", latitude, longitude)

	resp, err := http.Get(url)
	if err != nil {
		return execution.FuncResult{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return execution.FuncResult{}, err
	}

	var weatherData WeatherData
	if err := json.Unmarshal(body, &weatherData); err != nil {
		return execution.FuncResult{}, err
	}

	return execution.FuncResult{
		Present: true,
		Value: GetWeatherForecastResponse{
			Temperature2M: weatherData.Hourly.Temperature2M,
			WindSpeed10M:  weatherData.Hourly.WindSpeed10M,
		},
		FormatFunc: func() (string, error) {
			// Calculate statistics for temperature
			avgTemp := calculateAverage(weatherData.Hourly.Temperature2M)
			minTemp, maxTemp := findMinMax(weatherData.Hourly.Temperature2M)

			// Calculate statistics for wind speed
			avgWindSpeed := calculateAverage(weatherData.Hourly.WindSpeed10M)
			minWindSpeed, maxWindSpeed := findMinMax(weatherData.Hourly.WindSpeed10M)

			output := fmt.Sprintf("Here is the weather forecast for %f, %f:\n\n", latitude, longitude)
			output += fmt.Sprintf("Temperature Summary:\n- Average Temperature: %.1f°C\n- Minimum Temperature: %.1f°C\n- Maximum Temperature: %.1f°C\n\n", avgTemp, minTemp, maxTemp)
			output += fmt.Sprintf("Wind Speed Summary:\n- Average Wind Speed: %.1f km/h\n- Minimum Wind Speed: %.1f km/h\n- Maximum Wind Speed: %.1f km/h\n\n", avgWindSpeed, minWindSpeed, maxWindSpeed)
			return output, nil
		},
		Metadata: nil,
	}, nil
}

func argsToGetWeatherForecastRequest(args map[string]interface{}) (Coordinates, error) {
	if c, ok := args["coordinates"].(Coordinates); ok {
		return c, nil
	}

	jsonBytes, err := json.Marshal(args)
	if err != nil {
		return Coordinates{}, fmt.Errorf("error marshalling args to JSON: %w", err)
	}

	var req struct {
		Coordinates Coordinates `json:"coordinates"`
	}
	if err := json.Unmarshal(jsonBytes, &req); err != nil {
		return Coordinates{}, fmt.Errorf("error unmarshalling args: %w", err)
	}
	return req.Coordinates, nil
}

func calculateAverage(data []float64) float64 {
	var sum float64
	for _, value := range data {
		sum += value
	}
	return sum / float64(len(data))
}

func findMinMax(data []float64) (min, max float64) {
	min = math.MaxFloat64
	max = -math.MaxFloat64
	for _, value := range data {
		if value < min {
			min = value
		}
		if value > max {
			max = value
		}
	}
	return
}
