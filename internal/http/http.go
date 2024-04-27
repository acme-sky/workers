package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

type ResponseBody struct {
	Count uint                     `json:"count"`
	Data  []map[string]interface{} `json:"data"`
}

// Make a new request to an endpoint with a `body` and returns a response body
// or an error.
func MakeRequest(endpoint string, body map[string]interface{}) (*ResponseBody, error) {
	jsonBody, _ := json.Marshal(body)
	bodyReader := bytes.NewReader(jsonBody)

	req, err := http.NewRequest(http.MethodPost, endpoint, bodyReader)

	if err != nil {
		return nil, err
	}

	httpClient := http.Client{
		Timeout: 30 * time.Second,
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("HTTP request returned a status %d", res.StatusCode))
	}

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Could not read response body: %s", err.Error()))
	}

	var responseBody ResponseBody
	if err := json.Unmarshal(resBody, &responseBody); err != nil {
		return nil, errors.New(fmt.Sprintf("Could not unmarshal response body: %s", err))
	}

	return &responseBody, nil
}
