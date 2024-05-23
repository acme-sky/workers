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

// Service Task executed on "Activity_Foreach_AirlineService" loop in a case of
// "Any flight found?" = "Yes".
// It iterates all flights and save 'em as available.
func STSaveFlightsAsAvailable(client worker.JobClient, job entities.Job) {
	jobKey := job.GetKey()

	variables, err := job.GetVariablesAsMap()
	if err != nil {
		acmejob.FailJob(client, job)
		return
	}

	db, _ := db.GetDb()

	flights := variables["flights"].([]interface{})

	for i := 0; i < len(flights); i++ {
		flight := flights[i].(map[string]interface{})
		departure_airport := flight["departure_airport"].(map[string]interface{})
		flight["departure_airport"] = departure_airport["code"]
		arrival_airport := flight["arrival_airport"].(map[string]interface{})
		flight["arrival_airport"] = arrival_airport["code"]
		input, err := models.ValidateAvailableFlight(db, flight)

		if err != nil {
			log.Errorf("[%s] [%d] Error validating flight: %s", job.Type, jobKey, err.Error())
			continue
		}

		var available_flight models.AvailableFlight
		if found := db.Where("code = ? AND cost = ? AND departure_airport = ? AND arrival_airport = ? AND departure_time = ? AND arrival_time = ?",
			input.Code, input.Cost, input.DepartureAirport, input.ArrivalAirport, input.DepartureTime, input.ArrivalTime).First(&available_flight).Error; found == nil {
			log.Warnf("[%s] [%d] Skip an already saved flight", job.Type, jobKey)
			continue
		}

		new_available_flight := models.NewAvailableFlight(*input)

		if created := db.Create(&new_available_flight); created == nil {
			log.Errorf("[%s] [%d] Available flight not saved", job.Type, jobKey)
		} else {
			log.Infof("[%s] [%d] Available flight saved", job.Type, jobKey)
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

	acmejob.JobStatuses.Close(job.Type, 0)
}
