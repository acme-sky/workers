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

// Service Task raised by ACMESky Interests Manager lame every 1 hour.
// Get available flights info from the database and create journeys.
// by "Activity_Foreach_Journey".
func STCreateJourneys(client worker.JobClient, job entities.Job) {
	jobKey := job.GetKey()

	variables, err := job.GetVariablesAsMap()
	if err != nil {
		acmejob.FailJob(client, job)
		return
	}

	db, _ := db.GetDb()
	var available_flights []models.AvailableFlight

	if found := db.Where("departaure_time::date >= now()::date AND offer_sent = false").Preload("User").Preload("Interest").Find(&available_flights); found == nil {
		log.Errorf("[%s] [%d] Interests not found", job.Type, jobKey)
		acmejob.FailJob(client, job)
		return
	}

	interests := make(map[int][]models.AvailableFlight)

	var journeys []uint = []uint{}

	for _, flight := range available_flights {
		if flight.InterestId != nil {
			interests[*flight.InterestId] = append(interests[*flight.InterestId], flight)
		} else {
			in := map[string]interface{}{
				"flight1_id": flight.Id,
				"user_id":    flight.UserId,
				"cost":       flight.Cost,
			}
			input, err := models.ValidateJourney(db, in)

			if err != nil {
				log.Errorf("[%s] [%d] Error creating journey: %s", job.Type, jobKey, err.Error())
				acmejob.FailJob(client, job)
				return
			}

			journey := models.NewJourney(*input)
			if err := db.Create(&journey).Error; err != nil {
				log.Errorf("[%s] [%d] Journey not saved: %s", job.Type, jobKey, err.Error())
			} else {
				log.Infof("[%s] [%d] Journey saved", job.Type, jobKey)
				journeys = append(journeys, journey.Id)
			}
		}
	}

	for _, flights := range interests {
		var in map[string]interface{}

		if len(flights) == 2 {
			in = map[string]interface{}{
				"flight1_id": flights[0].Id,
				"flight2_id": flights[1].Id,
				"user_id":    flights[0].UserId,
				"cost":       flights[0].Cost + flights[1].Cost,
			}
		} else {
			in = map[string]interface{}{
				"flight1_id": flights[0].Id,
				"user_id":    flights[0].UserId,
				"cost":       flights[0].Cost,
			}
		}

		input, err := models.ValidateJourney(db, in)

		if err != nil {
			log.Errorf("[%s] [%d] Error creating journey: %s", job.Type, jobKey, err.Error())
			acmejob.FailJob(client, job)
			return
		}

		journey := models.NewJourney(*input)
		if err := db.Preload("AvailableFlight").Preload("User").Create(&journey).Error; err != nil {
			log.Errorf("[%s] [%d] Journey not saved: %s", job.Type, jobKey, err.Error())
		} else {
			log.Infof("[%s] [%d] Journey saved", job.Type, jobKey)
			journeys = append(journeys, journey.Id)
		}
	}

	variables["journeys"] = journeys

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
