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

type AuthTokenBody struct {
	Token string `json:"token"`
}

type JourneyResponseBody struct {
	Id              uint                   `json:"id"`
	CreatedAt       time.Time              `json:"created_at"`
	DepartureFlight map[string]interface{} `json:"departure_flight"`
	ArrivalFlight   map[string]interface{} `json:"arrival_flight"`
	Cost            float64                `json:"cost"`
	Email           string                 `json:"email"`
}

type PaymentResponseBody struct {
	Id          string  `json:"id"`
	Owner       string  `json:"owner"`
	Amount      float64 `json:"amount"`
	Description string  `json:"description"`
	Callback    string  `json:"callback"`
	Paid        bool    `json:"paid"`
	// We use string here instead of time.Time because we do not want to fix the
	// parsing error
	CreatedAt string `json:"created_at"`
}

type AirportInfoResponseBody struct {
	Id        uint    `json:"id"`
	Name      string  `json:"name"`
	Code      string  `json:"code"`
	Location  string  `json:"location"`
	Latitude  float32 `json:"latitude"`
	Longitude float32 `json:"longitude"`
}

type ProntogramMessageRequest struct {
	Message    string `json:"message"`
	Expiration string `json:"expiration"`
	Username   string `json:"username"`
	Sid        string `json:"sid"`
}

type ProntogramMessageResponse struct {
	Message string `json:"message"`
	Sid     string `json:"sid"`
	Status  int    `json:"status"`
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

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Could not read response body: %s", err.Error()))
	}

	if res.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("HTTP request returned a status %d and response `%s`", res.StatusCode, resBody))
	}

	var responseBody ResponseBody
	if err := json.Unmarshal(resBody, &responseBody); err != nil {
		return nil, errors.New(fmt.Sprintf("Could not unmarshal response body: %s", err))
	}

	return &responseBody, nil
}

// Make a login with credentials and returns the auth token
func MakeLogin(endpoint string, body map[string]interface{}) (*string, error) {
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

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Could not read response body: %s", err.Error()))
	}

	if res.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("HTTP request returned a status %d and response `%s`", res.StatusCode, resBody))
	}

	var responseBody AuthTokenBody
	if err := json.Unmarshal(resBody, &responseBody); err != nil {
		return nil, errors.New(fmt.Sprintf("Could not unmarshal response body: %s", err))
	}

	return &responseBody.Token, nil
}

// Make a new request to an endpoint with a `body` for a new journey. `auth` is
// a bearer token.
func NewJourneyRequest(endpoint string, body map[string]interface{}, auth string) (*JourneyResponseBody, error) {
	jsonBody, _ := json.Marshal(body)
	bodyReader := bytes.NewReader(jsonBody)

	req, err := http.NewRequest(http.MethodPost, endpoint, bodyReader)

	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", auth))

	httpClient := http.Client{
		Timeout: 30 * time.Second,
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Could not read response body: %s", err.Error()))
	}

	if res.StatusCode != 201 {
		return nil, errors.New(fmt.Sprintf("HTTP request returned a status %d and response `%s`", res.StatusCode, resBody))
	}

	var responseBody JourneyResponseBody
	if err := json.Unmarshal(resBody, &responseBody); err != nil {
		return nil, errors.New(fmt.Sprintf("Could not unmarshal response body: %s", err))
	}

	return &responseBody, nil
}

// Make a new request to an endpoint with a `body` for a new payment bank. `auth` is
// the API token.
func NewPaymentRequest(endpoint string, body map[string]interface{}, auth string) (*PaymentResponseBody, error) {
	jsonBody, _ := json.Marshal(body)
	bodyReader := bytes.NewReader(jsonBody)

	req, err := http.NewRequest(http.MethodPost, endpoint, bodyReader)

	if err != nil {
		return nil, err
	}

	req.Header.Add("X-API-TOKEN", auth)

	httpClient := http.Client{
		Timeout: 30 * time.Second,
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Could not read response body: %s", err.Error()))
	}

	if res.StatusCode != 201 {
		return nil, errors.New(fmt.Sprintf("HTTP request returned a status %d and response `%s`", res.StatusCode, resBody))
	}

	var responseBody PaymentResponseBody
	if err := json.Unmarshal(resBody, &responseBody); err != nil {
		return nil, errors.New(fmt.Sprintf("Could not unmarshal response body: %s", err))
	}

	return &responseBody, nil
}

// Make a new request to an endpoint to get info about an airport.
func GetAirportInfo(endpoint string) (*AirportInfoResponseBody, error) {
	res, err := http.Get(endpoint)
	if err != nil {
		return nil, err
	}

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Could not read response body: %s", err.Error()))
	}

	if res.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("HTTP request returned a status %d and response `%s`", res.StatusCode, resBody))
	}

	var responseBody AirportInfoResponseBody
	if err := json.Unmarshal(resBody, &responseBody); err != nil {
		return nil, errors.New(fmt.Sprintf("Could not unmarshal response body: %s", err))
	}

	return &responseBody, nil
}

// Make a new request for Prontogram and returns a ProntogramMessageResponse
func MakeProntogramRequest(endpoint string, body ProntogramMessageRequest) (*ProntogramMessageResponse, error) {
	jsonBody, _ := json.Marshal(body)
	bodyReader := bytes.NewReader(jsonBody)

	req, err := http.NewRequest(http.MethodPost, endpoint, bodyReader)

	if err != nil {
		return nil, err
	}

	httpClient := http.Client{
		Timeout: 30 * time.Second,
	}

	req.Header.Add("Content-Type", "application/json")

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Could not read response body: %s", err.Error()))
	}

	if res.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("HTTP request returned a status %d and response `%s`", res.StatusCode, resBody))
	}

	var responseBody ProntogramMessageResponse
	if err := json.Unmarshal(resBody, &responseBody); err != nil {
		return nil, errors.New(fmt.Sprintf("Could not unmarshal response body: %s", err))
	}

	return &responseBody, nil
}
