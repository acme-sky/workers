package http

import (
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

	params := gosoap.Params{
		"PickupAddress": *offer.User.Address,
		// FIXME: add "address" field to airport with string
		"Address":      offer.Journey.Flight1.DepartureAirport,
		"CustomerName": offer.User.Name,
		"PickupDate":   offer.Journey.Flight1.DepartureTime.Add(-2 * time.Hour).Format("02/01/2006 15:04"),
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
