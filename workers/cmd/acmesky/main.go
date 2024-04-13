package main

import (
	"os"
	"os/signal"

	"github.com/acme-sky/bpmn/workers/internal/acme"
	acmejob "github.com/acme-sky/bpmn/workers/internal/job"
)

var quit = make(chan os.Signal, 1)

func main() {
	client := acmejob.CreateClient("Process_ACME")
	defer (*client).Close()

	signal.Notify(quit, os.Interrupt)
	go func() {
		<-quit
		os.Exit(1)
	}()

	jobs := []acmejob.Job{
		// First part of User Profile lane
		{Name: "ST_Save_Flight", Handler: acme.STSaveFlight},
		{Name: "TM_Ack_Flight_Request_Save", Handler: acme.TMAckFlightRequestSave, Message: &acmejob.MessageCommand{Name: "CM_Ack_Flight_Request_Save", CorrelationKey: "0"}},

		// Interests manager lane
		{Name: "ST_Get_Interests", Handler: acme.STGetInterests},
		{Name: "ST_Check_Flight_For_An_Interest", Handler: acme.STCheckFlightForAnInterest},
		{Name: "ST_Prepare_Offer", Handler: acme.STPrepareOffer},
		{Name: "TM_Send_Offer", Handler: acme.TMSendOffer, Message: &acmejob.MessageCommand{Name: "CM_New_Message_For_Prontogram", CorrelationKey: "0"}},

		// User profile lane: check offer
		{Name: "ST_Retrieve_Offer", Handler: acme.STRetrieveOffer},
		{Name: "ST_Change_Offer_Status", Handler: acme.STChangeOfferStatus},
		{Name: "TM_Error_On_Check_Offer", Handler: acme.TMErrorOnCheckOffer, Message: &acmejob.MessageCommand{Name: "CM_Received_Bank_Error", CorrelationKey: "0"}},

		// User profile lane: book journey
		// Message fields for TM_Book_Journey and TM_Ask_Payment_Link is `nil` because it comunicates with an hidden participant
		{Name: "TM_Book_Journey", Handler: acme.TMBookJourney},
		{Name: "TM_Ask_Payment_Link", Handler: acme.TMAskPaymentLink, After: acme.TMAskPaymentLinkAfter},
		{Name: "TM_Send_Payment_Link", Handler: acme.TMSendPaymentLink, Message: &acmejob.MessageCommand{Name: "CM_Received_Bank_Link", CorrelationKey: "0"}},
		{Name: "ST_Offer_Still_Valid", Handler: acme.STOfferStillValid},
		{Name: "TM_Error_On_Book_Journey", Handler: acme.TMErrorOnBookJourney, Message: &acmejob.MessageCommand{Name: "CM_Received_Bank_Error", CorrelationKey: "0"}},
		{Name: "TM_Journey", Handler: acme.TMJourney, Message: &acmejob.MessageCommand{Name: "CM_Journey", CorrelationKey: "0"}},
		{Name: "TM_Computer_Distance_User_Airport", Handler: acme.TMComputerDistanceUserAirport},
		{Name: "TM_Find_Nearest_Available_Rent_Company", Handler: acme.TMFindNearestAvailableRentCompany},
		{Name: "TM_Ask_For_Rent", Handler: acme.TMAskForRent},
		{Name: "TM_Journey_And_Rent", Handler: acme.TMJourneyAndRent, Message: &acmejob.MessageCommand{Name: "CM_Journey_And_Rent", CorrelationKey: "0"}},
		{Name: "TM_Journey_Rent_Error", Handler: acme.TMJourneyRentError, Message: &acmejob.MessageCommand{Name: "CM_Journey", CorrelationKey: "0"}},

		// User profile lane: flights manager
		{Name: "ST_Get_User_Interests", Handler: acme.STGetUserInterests},
		{Name: "TM_Search_Flights_On_Airline", Handler: acme.TMSearchFlightsOnAirline},
		{Name: "ST_Save_Flights_As_Available", Handler: acme.STSaveFlightsAsAvailable},
	}

	for _, job := range jobs {
		go func(job *acmejob.Job) {
			job.Handle(client)
		}(&job)
	}

	<-quit
}
