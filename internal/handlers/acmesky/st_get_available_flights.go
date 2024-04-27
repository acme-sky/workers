package handlers

import (
	"context"

	"github.com/charmbracelet/log"

	"github.com/acme-sky/workers/internal/db"
	acmejob "github.com/acme-sky/workers/internal/job"
	"github.com/acme-sky/workers/internal/models"
	"github.com/camunda/zeebe/clients/go/v8/pkg/entities"
	"github.com/camunda/zeebe/clients/go/v8/pkg/worker"
)

// Service Task raised by ACMESky Interests Manager lame every 1 hour.
// Get available flights info from the database and save them in a new env variable read
// by "Activity_Foreach_Interest".
func STGetAvailableFlights(client worker.JobClient, job entities.Job) {
	jobKey := job.GetKey()

	variables, err := job.GetVariablesAsMap()
	if err != nil {
		acmejob.FailJob(client, job)
		return
	}

	db, _ := db.GetDb()
	var available_flights []models.AvailableFlight

	if found := db.Where("departaure_time::date >= now()::date AND offer_sent = false").Preload("User").Find(&available_flights); found == nil {
		log.Errorf("[%s] [%d] Interests not found", job.Type, jobKey)
		acmejob.FailJob(client, job)
		return
	}

	variables["available_flights"] = available_flights

	request, err := client.NewCompleteJobCommand().JobKey(jobKey).VariablesFromMap(variables)
	if err != nil {
		acmejob.FailJob(client, job)
		return
	}

	ctx := context.Background()
	_, err = request.Send(ctx)
	if err != nil {
		acmejob.FailJob(client, job)
		return
	}

	log.Infof("[%s] [%d] Successfully completed job", job.Type, jobKey)

	acmejob.JobVariables[job.Type] <- variables
	acmejob.JobStatuses.Close(job.Type, 0)
}
