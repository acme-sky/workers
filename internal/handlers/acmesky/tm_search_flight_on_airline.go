package handlers

import (
	"context"
	"github.com/charmbracelet/log"

	acmejob "github.com/acme-sky/workers/internal/job"
	"github.com/camunda/zeebe/clients/go/v8/pkg/entities"
	"github.com/camunda/zeebe/clients/go/v8/pkg/worker"
)

func TMSearchFlightsOnAirline(client worker.JobClient, job entities.Job) {
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

	variables["flight_is_found"] = true
	variables["flights"] = []map[string]interface{}{{"id": 6}, {"id": 12}}

	request, err := client.NewCompleteJobCommand().JobKey(jobKey).VariablesFromMap(variables)
	if err != nil {
		acmejob.FailJob(client, job)
		return
	}

	log.Debug("Processing data:", variables)

	ctx := context.Background()
	_, err = request.Send(ctx)
	if err != nil {
		acmejob.FailJob(client, job)
		return
	}

	log.Infof("Successfully completed job for %v", interest)
	acmejob.JobVariables[job.Type] <- variables
	acmejob.JobStatuses.Close(job.Type, 0)
}
