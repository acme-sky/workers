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

// Service Task raised by ACMESky Flights Manager lame every 1 hour.
// Get interests info from the database and save them in a new env variable read
// by "Activity_Foreach_AirlineService". Also, set up the airlines array used
// to iterate interests.
func STGetUserInterests(client worker.JobClient, job entities.Job) {
	jobKey := job.GetKey()

	variables, err := job.GetVariablesAsMap()
	if err != nil {
		acmejob.FailJob(client, job)
		return
	}

	db, _ := db.GetDb()
	var interests []models.Interest

	if found := db.Where("flight1_departure_time::date >= now()::date").Preload("User").Find(&interests); found == nil {
		log.Errorf("[%s] [%d] Interests not found", job.Type, jobKey)
		acmejob.FailJob(client, job)
		return
	}

	variables["interests"] = interests

	var airlines []models.Airline
	if err := db.Find(&airlines).Error; err != nil {
		panic("can't find airlines")
	}
	variables["airlines"] = airlines

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
