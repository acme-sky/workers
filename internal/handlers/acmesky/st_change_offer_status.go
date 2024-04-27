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

// Service Task raised when an offer token is valid.
// It changes its "is_used" to `true`.
func STChangeOfferStatus(client worker.JobClient, job entities.Job) {
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
	var offer models.Offer

	if err := db.Where("id = ?", int(variables["offer_id"].(float64))).First(&offer).Error; err != nil {
		log.Errorf("[%s] [%d] Error on getting offer %s", job.Type, jobKey, err.Error())
	}
	offer.IsUsed = true
	if err := db.Save(&offer).Error; err != nil {
		log.Errorf("[%s] [%d] Error on saving offer %s", job.Type, jobKey, err.Error())
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
