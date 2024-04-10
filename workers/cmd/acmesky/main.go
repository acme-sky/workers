package main

import (
	"time"

	"github.com/acme-sky/bpmn/workers/internal/acme"
	acmejob "github.com/acme-sky/bpmn/workers/internal/job"
)

func main() {
	client := acmejob.CreateClient("Process_ACME")
	defer (*client).Close()

	time.Sleep(10 * time.Second)

	jobs := []acmejob.Job{
		{Name: "CM_New_Request_Save_Flight", Handler: nil, Type_: acmejob.MessageType, CorrelationKey: "0"},
		{Name: "ST_Save_Flight", Handler: acme.STSaveFlight, Type_: acmejob.JobType, CorrelationKey: ""},
	}

	acmejob.HandleJobs(client, jobs)
}
