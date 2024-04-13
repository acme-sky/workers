package main

import (
	"os"
	"os/signal"

	acmejob "github.com/acme-sky/bpmn/workers/internal/job"
	"github.com/acme-sky/bpmn/workers/internal/user"
)

var quit = make(chan os.Signal, 1)

func main() {
	client := acmejob.CreateClient("Process_User")
	defer (*client).Close()

	signal.Notify(quit, os.Interrupt)
	go func() {
		<-quit
		os.Exit(1)
	}()

	jobs := []acmejob.Job{
		// First part when an user expresses interest to monitor a flight
		{Name: "TM_New_Request_Save_Flight", Handler: user.TMNewRequestSaveFlight, Message: &acmejob.MessageCommand{Name: "CM_New_Request_Save_Flight", CorrelationKey: "0"}},

		{Name: "TM_Check_Offer", Handler: user.TMCheckOffer, Message: &acmejob.MessageCommand{Name: "CM_Check_Offer", CorrelationKey: "0"}},
	}

	for _, job := range jobs {
		go func(job *acmejob.Job) {
			job.Handle(client)
		}(&job)
	}

	<-quit
}
