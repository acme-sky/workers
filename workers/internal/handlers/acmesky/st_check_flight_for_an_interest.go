package handlers

import (
	"context"
	"fmt"
	"github.com/charmbracelet/log"
	"math/rand"

	acmejob "github.com/acme-sky/bpmn/workers/internal/job"
	"github.com/camunda/zeebe/clients/go/v8/pkg/entities"
	"github.com/camunda/zeebe/clients/go/v8/pkg/worker"
)

func STCheckFlightForAnInterest(client worker.JobClient, job entities.Job) {
	jobKey := job.GetKey()

	variables, err := job.GetVariablesAsMap()
	if err != nil {
		acmejob.FailJob(client, job)
		return
	}

	interests := variables["interests"].([]interface{})
	index := int(variables["loopCounter"].(float64)) - 1

	if index < 0 || index >= len(interests) {
		panic("Index out of range")
	}
	interest := interests[index].(map[string]interface{})

	variables["flight"] = "y"
	r := rand.Int()
	if r%2 == 0 {
		variables["flight"] = nil
	}

	request, err := client.NewCompleteJobCommand().JobKey(jobKey).VariablesFromMap(variables)
	if err != nil {
		acmejob.FailJob(client, job)
		return
	}

	log.SetPrefix(fmt.Sprintf("[%s] [%d] ", job.Type, jobKey))

	log.Debug("Processing data:", variables)

	ctx := context.Background()
	_, err = request.Send(ctx)
	if err != nil {
		acmejob.FailJob(client, job)
		return
	}

	log.Infof("Successfully completed job for ", interest)
	acmejob.JobStatuses.Close(job.Type)
}
