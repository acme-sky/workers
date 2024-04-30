package main

import (
	"os"
	"os/signal"

	"github.com/acme-sky/workers/internal/config"
	"github.com/acme-sky/workers/internal/db"
	acmeskyHandlers "github.com/acme-sky/workers/internal/handlers/acmesky"
	prontogramHandlers "github.com/acme-sky/workers/internal/handlers/prontogram"
	userHandlers "github.com/acme-sky/workers/internal/handlers/user"
	acmejob "github.com/acme-sky/workers/internal/job"
	"github.com/acme-sky/workers/internal/message"
	"github.com/charmbracelet/log"
	"github.com/getsentry/sentry-go"
)

var quit = make(chan os.Signal, 1)

func main() {
	// Read environment variables and stops execution if any errors occur
	if err := config.LoadConfig(); err != nil {
		log.Printf("failed to load config. err %v", err)

		return
	}

	// Ignore error because if it failed on loading, it should raised an error
	// above.
	conf, _ := config.GetConfig()

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              conf.String("sentry.dsn"),
		TracesSampleRate: 0.7,
	})
	if err != nil {
		log.Errorf("sentry.Init: %s", err)
	}

	if _, err := db.InitDb(conf.String("database.dsn")); err != nil {
		log.Fatalf("failed to connect database. err %v", err)

		return
	}

	client := acmejob.CreateClient(conf.String("process.id"))
	defer (*client).Close()

	signal.Notify(quit, os.Interrupt)
	go func() {
		<-quit
		os.Exit(1)
	}()

	go func() {
		message.MessageBroker(client)
	}()

	jobs := []acmejob.Job{
		// ------------- USER -------------
		// First part when an user expresses interest to monitor a flight
		{Name: "TM_New_Request_Save_Flight", Handler: userHandlers.TMNewRequestSaveFlight, Message: &acmejob.MessageCommand{Name: "CM_New_Request_Save_Flight", CorrelationKey: "0"}},
		{Name: "TM_Check_Offer", Handler: userHandlers.TMCheckOffer, Message: &acmejob.MessageCommand{Name: "CM_Check_Offer", CorrelationKey: "0"}},

		// ------------- PRONTOGRAM -------------
		{Name: "ST_Save_Info_On_Prontogram", Handler: prontogramHandlers.STSaveInfoOnProntogram, Message: nil},
		{Name: "TM_Propagate_Message_From_Prontogram", Handler: prontogramHandlers.TMPropagateMessageFromProntogram, Message: &acmejob.MessageCommand{Name: "Start_Received_New_Offer", CorrelationKey: "0"}},

		// ------------- ACMESKY -------------
		// First part of User Profile lane
		{Name: "ST_Save_Flight", Handler: acmeskyHandlers.STSaveFlight},
		{Name: "TM_Ack_Flight_Request_Save", Handler: acmeskyHandlers.TMAckFlightRequestSave, Message: &acmejob.MessageCommand{Name: "CM_Ack_Flight_Request_Save", CorrelationKey: "0"}},

		// Interests manager lane
		{Name: "ST_Create_Journeys", Handler: acmeskyHandlers.STCreateJourneys},
		{Name: "ST_Prepare_Offer", Handler: acmeskyHandlers.STPrepareOffer},
		{Name: "TM_Send_Offer", Handler: acmeskyHandlers.TMSendOffer, Message: &acmejob.MessageCommand{Name: "CM_New_Message_For_Prontogram", CorrelationKey: "0"}},

		// User profile lane: check offer
		{Name: "ST_Retrieve_Offer", Handler: acmeskyHandlers.STRetrieveOffer},
		{Name: "ST_Change_Offer_Status", Handler: acmeskyHandlers.STChangeOfferStatus},
		{Name: "TM_Error_On_Check_Offer", Handler: acmeskyHandlers.TMErrorOnCheckOffer, Message: &acmejob.MessageCommand{Name: "CM_Received_Bank_Error", CorrelationKey: "0"}},

		// User profile lane: book journey
		// Message fields for TM_Book_Journey and TM_Ask_Payment_Link is `nil` because it comunicates with an hidden participant
		{Name: "TM_Book_Journey", Handler: acmeskyHandlers.TMBookJourney},
		{Name: "TM_Ask_Payment_Link", Handler: acmeskyHandlers.TMAskPaymentLink},
		{Name: "TM_Send_Payment_Link", Handler: acmeskyHandlers.TMSendPaymentLink, Message: &acmejob.MessageCommand{Name: "CM_Received_Bank_Link", CorrelationKey: "0"}},
		{Name: "ST_Offer_Still_Valid", Handler: acmeskyHandlers.STOfferStillValid},
		{Name: "TM_Error_On_Book_Journey", Handler: acmeskyHandlers.TMErrorOnBookJourney, Message: &acmejob.MessageCommand{Name: "CM_Received_Bank_Error", CorrelationKey: "0"}},
		{Name: "TM_Journey", Handler: acmeskyHandlers.TMJourney, Message: &acmejob.MessageCommand{Name: "CM_Journey", CorrelationKey: "0"}},
		{Name: "TM_Computer_Distance_User_Airport", Handler: acmeskyHandlers.TMComputerDistanceUserAirport},
		{Name: "TM_Find_Nearest_Available_Rent_Company", Handler: acmeskyHandlers.TMFindNearestAvailableRentCompany},
		{Name: "TM_Ask_For_Rent", Handler: acmeskyHandlers.TMAskForRent},
		{Name: "TM_Journey_And_Rent", Handler: acmeskyHandlers.TMJourneyAndRent, Message: &acmejob.MessageCommand{Name: "CM_Journey_And_Rent", CorrelationKey: "0"}},
		{Name: "TM_Journey_Rent_Error", Handler: acmeskyHandlers.TMJourneyRentError, Message: &acmejob.MessageCommand{Name: "CM_Journey", CorrelationKey: "0"}},

		// User profile lane: flights manager
		{Name: "ST_Save_Last_Minute_Offer", Handler: acmeskyHandlers.STSaveLastMinuteOffer},
		{Name: "ST_Get_User_Interests", Handler: acmeskyHandlers.STGetUserInterests},
		{Name: "TM_Search_Flights_On_Airline", Handler: acmeskyHandlers.TMSearchFlightsOnAirline},
		{Name: "ST_Save_Flights_As_Available", Handler: acmeskyHandlers.STSaveFlightsAsAvailable},
	}

	for _, job := range jobs {
		go func(job *acmejob.Job) {
			job.Handle(client)
		}(&job)
	}

	<-quit
}
