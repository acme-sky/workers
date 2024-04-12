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
		{Name: "ST_Save_Flight", Handler: acme.STSaveFlight, Message: nil},
		{Name: "TM_Ack_Flight_Request_Save", Handler: acme.TMAckFlightRequestSave, Message: &acmejob.MessageCommand{Name: "CM_Ack_Flight_Request_Save", CorrelationKey: "0"}},

		{Name: "ST_Get_Interests", Handler: acme.STGetInterests, Message: nil},
		{Name: "ST_Check_Flight_For_An_Interest", Handler: acme.STCheckFlightForAnInterest, Message: nil},
		{Name: "ST_Prepare_Offer", Handler: acme.STPrepareOffer, Message: nil},
		{Name: "TM_Send_Offer", Handler: acme.TMSendOffer, Message: &acmejob.MessageCommand{Name: "CM_New_Message_For_Prontogram", CorrelationKey: "0"}},
		{Name: "ST_Retrieve_Offer", Handler: acme.STRetrieveOffer, Message: nil},
		{Name: "ST_Change_Offer_Status", Handler: acme.STChangeOfferStatus, Message: nil},
		{Name: "TM_Error_On_Check_Offer", Handler: acme.TMErrorOnCheckOffer, Message: &acmejob.MessageCommand{Name: "CM_Received_Bank_Error", CorrelationKey: "0"}},
	}

	for _, job := range jobs {
		go func(job *acmejob.Job) {
			acmejob.HandleJob(client, *job)
		}(&job)
	}

	<-quit
}
