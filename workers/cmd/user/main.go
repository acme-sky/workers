package main

import (
	acmejob "github.com/acme-sky/bpmn/workers/internal/job"
	"github.com/acme-sky/bpmn/workers/internal/user"
)

func main() {
	client := acmejob.CreateClient("Process_User")
	defer (*client).Close()

	jobs := []acmejob.Job{
		{Name: "TM_New_Request_Save_Flight", Handler: user.TMNewRequestSaveFlight, Type_: acmejob.JobType, CorrelationKey: ""},
		{Name: "CM_Ack_Flight_Request_Save", Handler: nil, Type_: acmejob.MessageType, CorrelationKey: "0"},
	}

	acmejob.HandleJobs(client, jobs)
}
