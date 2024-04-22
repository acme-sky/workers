package main

import (
	"os"
	"os/signal"

	handlers "github.com/acme-sky/bpmn/workers/internal/handlers/acmesky"
	acmejob "github.com/acme-sky/bpmn/workers/internal/job"
	"github.com/acme-sky/bpmn/workers/pkg/message"
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

	go func() {
		message.MessageBroker()
	}()

	jobs := []acmejob.Job{
		// First part of User Profile lane
		{Name: "ST_Save_Flight", Handler: handlers.STSaveFlight},
		{Name: "TM_Ack_Flight_Request_Save", Handler: handlers.TMAckFlightRequestSave, Message: &acmejob.MessageCommand{Name: "CM_Ack_Flight_Request_Save", CorrelationKey: "0"}},

		// Interests manager lane
		{Name: "ST_Get_Interests", Handler: handlers.STGetInterests},
		{Name: "ST_Check_Flight_For_An_Interest", Handler: handlers.STCheckFlightForAnInterest},
		{Name: "ST_Prepare_Offer", Handler: handlers.STPrepareOffer},
		{Name: "TM_Send_Offer", Handler: handlers.TMSendOffer, Message: &acmejob.MessageCommand{Name: "CM_New_Message_For_Prontogram", CorrelationKey: "0"}},

		// User profile lane: check offer
		{Name: "ST_Retrieve_Offer", Handler: handlers.STRetrieveOffer},
		{Name: "ST_Change_Offer_Status", Handler: handlers.STChangeOfferStatus},
		{Name: "TM_Error_On_Check_Offer", Handler: handlers.TMErrorOnCheckOffer, Message: &acmejob.MessageCommand{Name: "CM_Received_Bank_Error", CorrelationKey: "0"}},

		// User profile lane: book journey
		// Message fields for TM_Book_Journey and TM_Ask_Payment_Link is `nil` because it comunicates with an hidden participant
		{Name: "TM_Book_Journey", Handler: handlers.TMBookJourney},
		{Name: "TM_Ask_Payment_Link", Handler: handlers.TMAskPaymentLink, After: handlers.TMAskPaymentLinkAfter},
		{Name: "TM_Send_Payment_Link", Handler: handlers.TMSendPaymentLink, Message: &acmejob.MessageCommand{Name: "CM_Received_Bank_Link", CorrelationKey: "0"}},
		{Name: "ST_Offer_Still_Valid", Handler: handlers.STOfferStillValid},
		{Name: "TM_Error_On_Book_Journey", Handler: handlers.TMErrorOnBookJourney, Message: &acmejob.MessageCommand{Name: "CM_Received_Bank_Error", CorrelationKey: "0"}},
		{Name: "TM_Journey", Handler: handlers.TMJourney, Message: &acmejob.MessageCommand{Name: "CM_Journey", CorrelationKey: "0"}},
		{Name: "TM_Computer_Distance_User_Airport", Handler: handlers.TMComputerDistanceUserAirport},
		{Name: "TM_Find_Nearest_Available_Rent_Company", Handler: handlers.TMFindNearestAvailableRentCompany},
		{Name: "TM_Ask_For_Rent", Handler: handlers.TMAskForRent},
		{Name: "TM_Journey_And_Rent", Handler: handlers.TMJourneyAndRent, Message: &acmejob.MessageCommand{Name: "CM_Journey_And_Rent", CorrelationKey: "0"}},
		{Name: "TM_Journey_Rent_Error", Handler: handlers.TMJourneyRentError, Message: &acmejob.MessageCommand{Name: "CM_Journey", CorrelationKey: "0"}},

		// User profile lane: flights manager
		{Name: "ST_Get_User_Interests", Handler: handlers.STGetUserInterests},
		{Name: "TM_Search_Flights_On_Airline", Handler: handlers.TMSearchFlightsOnAirline},
		{Name: "ST_Save_Flights_As_Available", Handler: handlers.STSaveFlightsAsAvailable},
	}

	for _, job := range jobs {
		go func(job *acmejob.Job) {
			job.Handle(client)
		}(&job)
	}

	<-quit
}
