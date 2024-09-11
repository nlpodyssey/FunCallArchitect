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
	"net/http"
	"net/url"
	"strconv"

	"github.com/nlpodyssey/funcallarchitect/execution"
	"github.com/nlpodyssey/funcallarchitect/progress"
)

// Coordinates represents the response structure for geocoding
type Coordinates struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

func (t *Tools) GetCoordinates(_ context.Context, args map[string]interface{}, progress progress.Stream) (execution.FuncResult, error) {
	city, ok := args["city"].(string)
	if !ok {
		return execution.FuncResult{}, fmt.Errorf("city argument is required")
	}

	progress.Send("Retrieving coordinates for...")
	u := fmt.Sprintf("https://nominatim.openstreetmap.org/search?q=%s&format=json", url.QueryEscape(city))

	resp, err := http.Get(u)
	if err != nil {
		return execution.FuncResult{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return execution.FuncResult{}, fmt.Errorf("failed to retrieve coordinates for %s: %s", city, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return execution.FuncResult{}, err
	}

	type NominatimResponse []struct {
		Lat string `json:"lat"`
		Lon string `json:"lon"`
	}

	var nominatimResp NominatimResponse
	if err := json.Unmarshal(body, &nominatimResp); err != nil {
		return execution.FuncResult{}, err
	}

	if len(nominatimResp) > 0 {
		latitude := nominatimResp[0].Lat
		longitude := nominatimResp[0].Lon

		lat, _ := strconv.ParseFloat(latitude, 64)
		lon, _ := strconv.ParseFloat(longitude, 64)

		return execution.FuncResult{
			Present: true,
			Value: Coordinates{
				Lat: lat,
				Lon: lon,
			},
			FormatFunc: func() (string, error) {
				return fmt.Sprintf("Latitude: %f, Longitude: %f", lat, lon), nil
			},
			Metadata: nil,
		}, nil
	}

	return execution.FuncResult{
		Present: false,
		FormatFunc: func() (string, error) {
			return "Location not found", nil
		},
		Metadata: nil,
	}, nil
}
