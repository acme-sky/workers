package handlers

import (
	"context"

	"github.com/charmbracelet/log"

	"github.com/acme-sky/workers/internal/db"
	"github.com/acme-sky/workers/internal/http"
	acmejob "github.com/acme-sky/workers/internal/job"
	"github.com/acme-sky/workers/internal/models"
	"github.com/camunda/zeebe/clients/go/v8/pkg/entities"
	"github.com/camunda/zeebe/clients/go/v8/pkg/worker"
)

// Task used to create a new rent for an offer
func TMAskForRent(client worker.JobClient, job entities.Job) {
	jobKey := job.GetKey()

	variables, err := job.GetVariablesAsMap()
	if err != nil {
		acmejob.FailJob(client, job)
		return
	}

	db, _ := db.GetDb()
	var rent models.Rent

	rentCompanies := variables["rent_companies"].([]interface{})
	index := int(variables["loopCounter"].(float64)) - 1

	if len(rentCompanies) == 0 {
		log.Infof("[%s] [%d] You must define a rent_company object", job.Type, jobKey)
		acmejob.FailJob(client, job)
		return
	}
	rentCompany := rentCompanies[index].(map[string]interface{})
	rentCompanyId := int(rentCompany["Id"].(float64))

	if err := db.Where("id = ?", rentCompanyId).First(&rent).Error; err != nil {
		log.Errorf("[%s] [%d] Rent not found %s", job.Type, jobKey, err.Error())
		acmejob.FailJob(client, job)
		return
	}

	var offer models.Offer
	if err := db.Where("id = ?", variables["offer_id"]).Preload("Journey").Preload("Journey.Flight1").Preload("Journey.Flight2").Preload("User").First(&offer).Error; err != nil {
		log.Errorf("[%s] [%d] Journey not found", job.Type, jobKey)
		acmejob.FailJob(client, job)
		return
	}

	response, err := http.MakeRentRequest(rent, offer)

	if err != nil {
		log.Errorf("[%s] [%d] Error for rent `%s`: %s", job.Type, jobKey, rent.Name, err.Error())
		acmejob.FailJob(client, job)
		return
	} else {
		if response.Status == "OK" {
			variables["rent_status"] = "Ok"
			offer.RentId = response.RentId
			if err := db.Save(&offer).Error; err != nil {
				log.Errorf("[%s] [%d] Error on saving offer %s", job.Type, jobKey, err.Error())
				acmejob.FailJob(client, job)
				return
			}
			log.Infof("[%s] [%d] Rent `%s` is OK with ID `%s`", job.Type, jobKey, rent.Name, response.RentId)
		} else {
			log.Errorf("[%s] [%d] Rent `%s` is not OK", job.Type, jobKey, rent.Name)
		}
	}

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

	log.Infof("[%s] [%d] Successfully completed job", job.Type, jobKey)
	acmejob.JobVariables[job.Type] <- variables

	acmejob.JobStatuses.Close(job.Type, 0)
}
