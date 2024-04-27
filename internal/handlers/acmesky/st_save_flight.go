package handlers

import (
	"context"
	"fmt"

	"github.com/charmbracelet/log"

	"github.com/acme-sky/workers/internal/db"
	acmejob "github.com/acme-sky/workers/internal/job"
	"github.com/acme-sky/workers/internal/models"
	"github.com/camunda/zeebe/clients/go/v8/pkg/entities"
	"github.com/camunda/zeebe/clients/go/v8/pkg/worker"
)

// Service Task raised by ACMESky Profile lame.
// Save flight interest for an user.
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

	log.SetPrefix(fmt.Sprintf("[%s] [%d] ", job.Type, jobKey))

	db, _ := db.GetDb()

	input, err := models.ValidateInterest(db, variables)

	if err != nil {
		log.Errorf("Error validating interest: %s", err.Error())
		acmejob.FailJob(client, job)
		return
	}

	interest := models.NewInterest(*input)

	if created := db.Create(&interest); created == nil {
		log.Errorf("Interest not saved")
		acmejob.FailJob(client, job)
		return
	}

	ctx := context.Background()
	_, err = request.Send(ctx)
	if err != nil {
		acmejob.FailJob(client, job)
		return
	}

	log.Infof("Successfully completed job")

	acmejob.JobStatuses.Close(job.Type, 0)
}
