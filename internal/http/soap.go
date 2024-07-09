package http

import (
	"fmt"
	"net/http"
	"time"

	"github.com/acme-sky/workers/internal/models"
	"github.com/charmbracelet/log"
	"github.com/tiaguinho/gosoap"
)

type BookRentResult struct {
	Response BookRentResponse `xml:"BookRentResponse"`
}

type BookRentResponse struct {
	Status string `xml:"Status"`
	RentId string `xml:"RentId"`
}

type GetRentByIdResult struct {
	Response GetRentByIdResponse `xml:"GetRentByIdResponse"`
}

type GetRentByIdResponse struct {
	Status        string `xml:"Status"`
	RentId        string `xml:"RentId"`
	PickupAddress string `xml:"PickupAddress"`
	Address       string `xml:"Address"`
	CustomerName  string `xml:"CustomerName"`
	PickupDate    string `xml:"PickupDate"`
}

// SOAP call to BookRent action for a selected rent. Returns the call response
// which has a Status and RentId, the latter will be saved on the offer journey
func MakeRentRequest(rent models.Rent, offer models.Offer) (*BookRentResponse, error) {
	httpClient := &http.Client{
		Timeout: 1500 * time.Millisecond,
	}
	soap, err := gosoap.SoapClient(rent.Endpoint, httpClient)
	if err != nil {
		log.Errorf("SoapClient error: %s", err)
		return nil, err
	}

	endpoint := fmt.Sprintf("%s/airports/code/%s/", offer.Journey.Flight1.Airline, offer.Journey.Flight1.DepartureAirport)
	airport, err := GetAirportInfo(endpoint)
	if err != nil {
		log.Errorf("Can't find info for departure airport: %s", err.Error())
		return nil, err
	}

	params := gosoap.Params{
		"PickupAddress": *offer.User.Address,
		"Address":       airport.Location,
		"CustomerName":  offer.User.Name,
		"PickupDate":    offer.Journey.Flight1.DepartureTime.Add(-2 * time.Hour).Format("02/01/2006 15:04"),
	}

	res, err := soap.Call("BookRent", params)
	if err != nil {
		log.Fatalf("Call error: %s", err)
		return nil, err
	}

	var r BookRentResponse
	res.Unmarshal(&r)

	return &r, nil
}

// SOAP call to GetRentById action for a selected rent. Returns the reservation
// object data
func MakeGetRentByIdRequest(endpoint string, id string) (*GetRentByIdResponse, error) {
	httpClient := &http.Client{
		Timeout: 1500 * time.Millisecond,
	}
	soap, err := gosoap.SoapClient(endpoint, httpClient)
	if err != nil {
		log.Errorf("SoapClient error: %s", err)
		return nil, err
	}

	params := gosoap.Params{
		"RentId": id,
	}

	res, err := soap.Call("GetRentById", params)
	if err != nil {
		log.Fatalf("Call error: %s", err)
		return nil, err
	}

	var r GetRentByIdResponse
	res.Unmarshal(&r)

	return &r, nil
}
