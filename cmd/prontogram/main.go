package main

import (
	"os"
	"os/signal"

	acmejob "github.com/acme-sky/workers/internal/job"
	handlers "github.com/acme-sky/workers/internal/handlers/prontogram"
)

var quit = make(chan os.Signal, 1)

func main() {
	client := acmejob.CreateClient("Process_Prontogram")
	defer (*client).Close()

	signal.Notify(quit, os.Interrupt)
	go func() {
		<-quit
		os.Exit(1)
	}()

	jobs := []acmejob.Job{
		{Name: "ST_Save_Info_On_Prontogram", Handler: handlers.STSaveInfoOnProntogram, Message: nil},
		{Name: "TM_Propagate_Message_From_Prontogram", Handler: handlers.TMPropagateMessageFromProntogram, Message: &acmejob.MessageCommand{Name: "Start_Received_New_Offer", CorrelationKey: "0"}},
	}

	for _, job := range jobs {
		go func(job *acmejob.Job) {
			job.Handle(client)
		}(&job)
	}

	<-quit
}
