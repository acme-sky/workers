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

// Service Task raised when an airline sends a "last minute" offer. It creates
// an available flight to every user.
func STSaveLastMinuteOffer(client worker.JobClient, job entities.Job) {
	jobKey := job.GetKey()

	variables, err := job.GetVariablesAsMap()
	if err != nil {
		acmejob.FailJob(client, job)
		return
	}

	db, _ := db.GetDb()

	var users []models.User

	flight := variables["flight"].(map[string]interface{})

	db.Find(&users)

	countSaved := 0
	countNotSaved := 0
	for _, user := range users {
		flight["user_id"] = user.ID
		input, err := models.ValidateAvailableFlight(db, flight)

		if err != nil {
			log.Errorf("[%s] [%d] Error validating flight: %s", job.Type, jobKey, err.Error())
			acmejob.FailJob(client, job)
			return
		}

		var available_flight models.AvailableFlight
		if found := db.Where("code = ? AND cost = ? AND departure_airport = ? AND arrival_airport = ? AND departure_time = ? AND arrival_time = ?",
			input.Code, input.Cost, input.DepartureAirport, input.ArrivalAirport, input.DepartureTime, input.ArrivalTime).First(&available_flight).Error; found == nil {
			log.Warnf("[%s] [%d] Skip an already saved flight", job.Type, jobKey)
			countNotSaved++
			continue
		}

		new_available_flight := models.NewAvailableFlight(*input)

		if err := db.Create(&new_available_flight).Error; err != nil {
			log.Errorf("[%s] [%d] Available flight not saved: %s", job.Type, jobKey, err.Error())
			countNotSaved++
		} else {
			log.Infof("[%s] [%d] Available flight saved", job.Type, jobKey)
			countSaved++
		}
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
	log.Infof("[%s] [%d] Created %d available flights and %d ignored", job.Type, jobKey, countSaved, countNotSaved)

	acmejob.JobStatuses.Close(job.Type, 0)
}
