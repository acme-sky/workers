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

// Make a message request to the user for "journey invoice"
func TMInvoice(client worker.JobClient, job entities.Job) {
	jobKey := job.GetKey()

	variables, err := job.GetVariablesAsMap()
	if err != nil {
		acmejob.FailJob(client, job)
		return
	}

	db, _ := db.GetDb()
	var offer models.Offer
	if err := db.Where("id = ?", variables["offer_id"]).Preload("Journey").Preload("Journey.Flight1").Preload("Journey.Flight2").Preload("User").First(&offer).Error; err != nil {
		log.Errorf("[%s] [%d] Offer not found", job.Type, jobKey)
		acmejob.FailJob(client, job)
		return
	}

	offer.PaymentPaid = true
	if err := db.Save(&offer).Error; err != nil {
		log.Errorf("[%s] [%d] Error on saving offer %s", job.Type, jobKey, err.Error())
		acmejob.FailJob(client, job)
		return
	}

	invoice := models.NewInvoice(models.InvoiceInput{
		JourneyId: offer.JourneyId,
		UserId:    offer.UserId,
	})
	if created := db.Create(&invoice); created == nil {
		log.Errorf("[%s] [%d] Invoice not saved", job.Type, jobKey)
		acmejob.FailJob(client, job)
		return
	} else {
		log.Infof("[%s] [%d] Invoice saved", job.Type, jobKey)
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
	acmejob.JobVariables[job.Type] <- variables

	acmejob.JobStatuses.Close(job.Type, 0)
}
