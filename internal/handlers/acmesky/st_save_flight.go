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

// Service Task raised by ACMESky Profile lame.
// Save flight interest for an user. It wants a payload like:
//
// {
// "departaure_airport": "CTA",
// "departuare_time":    "2024-04-26T21:50:00Z",
// "arrival_airport":    "CPH",
// "arrival_time":       "2024-04-27T01:50:00Z",
// "user_id":            1,
// }
func STSaveFlight(client worker.JobClient, job entities.Job) {
	jobKey := job.GetKey()

	variables, err := job.GetVariablesAsMap()
	if err != nil {
		acmejob.FailJob(client, job)
		return
	}

	request, err := client.NewCompleteJobCommand().JobKey(jobKey).VariablesFromMap(variables)
	if err != nil {
		acmejob.FailJob(client, job)
		return
	}

	db, _ := db.GetDb()

	input, err := models.ValidateInterest(db, variables)

	if err != nil {
		log.Errorf("[%s] [%d] Error validating interest: %s", job.Type, jobKey, err.Error())
		acmejob.FailJob(client, job)
		return
	}

	interest := models.NewInterest(*input)

	if created := db.Create(&interest); created == nil {
		log.Errorf("[%s] [%d] Interest not saved", job.Type, jobKey)
		acmejob.FailJob(client, job)
		return
	} else {
		log.Infof("[%s] [%d] Interest saved", job.Type, jobKey)
	}

	ctx := context.Background()
	_, err = request.Send(ctx)
	if err != nil {
		acmejob.FailJob(client, job)
		return
	}

	log.Infof("[%s] [%d] Successfully completed job", job.Type, jobKey)

	acmejob.JobStatuses.Close(job.Type, 0)
}
