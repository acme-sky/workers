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

// Service Task raised by ACMESky when an user sends an offer token.
// It Checks if the offer is valid for this `token` variable.
func STRetrieveOffer(client worker.JobClient, job entities.Job) {
	jobKey := job.GetKey()

	variables, err := job.GetVariablesAsMap()
	if err != nil {
		acmejob.FailJob(client, job)
		return
	}

	db, _ := db.GetDb()

	var offer models.Offer

	if err := db.Where("token = ? AND is_used = 'f' AND to_timestamp(expired::double precision) >= current_timestamp", variables["token"]).First(&offer).Error; err != nil {
		log.Errorf("[%s] [%d] Token `%s` is not valid", job.Type, jobKey, variables["token"])
		variables["offer_id"] = nil
	} else {
		variables["offer_id"] = offer.Id
	}

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
	acmejob.JobStatuses.Close(job.Type, 0)
}
