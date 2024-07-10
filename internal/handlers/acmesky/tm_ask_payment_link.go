package handlers

import (
	"context"
	"fmt"

	"github.com/charmbracelet/log"

	"github.com/acme-sky/workers/internal/config"
	"github.com/acme-sky/workers/internal/db"
	"github.com/acme-sky/workers/internal/http"
	acmejob "github.com/acme-sky/workers/internal/job"
	"github.com/acme-sky/workers/internal/models"
	"github.com/camunda/zeebe/clients/go/v8/pkg/entities"
	"github.com/camunda/zeebe/clients/go/v8/pkg/worker"
)

// Task who creates a new payment link for an offer.
func TMAskPaymentLink(client worker.JobClient, job entities.Job) {
	jobKey := job.GetKey()

	variables, err := job.GetVariablesAsMap()
	if err != nil {
		acmejob.FailJob(client, job)
		return
	}

	conf, _ := config.GetConfig()

	db, _ := db.GetDb()
	var offer models.Offer
	if err := db.Where("id = ?", variables["offer_id"]).Preload("Journey").Preload("Journey.Flight1").Preload("Journey.Flight2").Preload("User").First(&offer).Error; err != nil {
		log.Errorf("[%s] [%d] Offer not found", job.Type, jobKey)
		acmejob.FailJob(client, job)
		return
	}

	endpoint := fmt.Sprintf("%s/payments/", conf.String("bank.endpoint"))
	payload := map[string]interface{}{
		"owner":    fmt.Sprintf("%s <%s>", offer.User.Name, offer.User.Email),
		"amount":   offer.Journey.Cost,
		"callback": fmt.Sprintf("%s/%d/", conf.String("bank.callback"), offer.Id),
	}

	if offer.Journey.Flight2 != nil {
		payload["description"] = fmt.Sprintf("Flights from %s to %s and from %s to %s",
			offer.Journey.Flight1.DepartureAirport,
			offer.Journey.Flight1.ArrivalAirport,
			offer.Journey.Flight2.DepartureAirport,
			offer.Journey.Flight2.ArrivalAirport)
	} else {
		payload["description"] = fmt.Sprintf("Flight from %s to %s",
			offer.Journey.Flight1.DepartureAirport,
			offer.Journey.Flight1.ArrivalAirport)
	}

	response, err := http.NewPaymentRequest(endpoint, payload, conf.String("bank.token"))

	if err != nil {
		log.Errorf("[%s] [%d] Error for offer `%d`: %s", job.Type, jobKey, offer.Id, err.Error())
		acmejob.FailJob(client, job)
		return
	}

	variables["payment_link"] = fmt.Sprintf("%s%s", conf.String("bank.payment.endpoint"), response.Id)
	variables["flight_price"] = offer.Journey.Cost

	offer.PaymentLink = variables["payment_link"].(string)
	if err := db.Save(&offer).Error; err != nil {
		log.Errorf("[%s] [%d] Error on saving offer %s", job.Type, jobKey, err.Error())
		acmejob.FailJob(client, job)
		return
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
