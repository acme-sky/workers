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

	if variables["rent_company"] == nil {
		log.Infof("[%s] [%d] You must define a rent_company object", job.Type, jobKey)
		acmejob.FailJob(client, job)
		return
	}

	rentCompany := variables["rent_company"].(map[string]interface{})
	rentCompanyId := int(rentCompany["id"].(float64))
	if err := db.Where("id = ?", rentCompanyId).First(&rent).Error; err != nil {
		log.Errorf("[%s] [%d] Rent not found %s", job.Type, jobKey, err.Error())
		acmejob.FailJob(client, job)
		return
	}
	variables["next_rent_company_to_check"] = 42

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
