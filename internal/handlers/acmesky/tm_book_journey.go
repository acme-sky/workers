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

// Task used to book a journey in an airline company. It first checks if the
// flight still exists and then, after a login to the airline company, makes the
// request for saving the journey.
func TMBookJourney(client worker.JobClient, job entities.Job) {
	jobKey := job.GetKey()

	variables, err := job.GetVariablesAsMap()
	if err != nil {
		acmejob.FailJob(client, job)
		return
	}

	db, _ := db.GetDb()
	var offer models.Offer
	if err := db.Where("id = ?", variables["offer_id"]).Preload("Journey").Preload("Journey.Flight1").Preload("Journey.Flight2").Preload("User").First(&offer).Error; err != nil {
		log.Errorf("[%s] [%d] Journey not found", job.Type, jobKey)
		acmejob.FailJob(client, job)
		return
	}

	flight1 := offer.Journey.Flight1
	// This flight ID refers to the airline flight ID
	var flight1_id int = 0

	flight2 := offer.Journey.Flight2
	// This flight ID refers to the airline flight ID
	var flight2_id int = 0

	endpoint := fmt.Sprintf("%s/flights/filter/", flight1.Airline)
	payload := map[string]interface{}{
		"code":              flight1.Code,
		"departure_airport": flight1.DepartureAirport,
		"departure_time":    flight1.DepartureTime,
		"arrival_airport":   flight1.ArrivalAirport,
		"arrival_time":      flight1.ArrivalTime,
	}

	response, err := http.MakeRequest(endpoint, payload)

	if err != nil {
		log.Errorf("[%s] [%d] Error for airline `%s`: %s", job.Type, jobKey, flight1.Airline, err.Error())
		acmejob.FailJob(client, job)
		return
	} else {
		if response.Count > 0 {
			if response.Count == 1 {
				flight1_id = int(response.Data[0]["id"].(float64))
			} else {
				log.Errorf("[%s] [%d] Found `%d` flights for flight1 = `%d`", job.Type, jobKey, response.Count, flight1.Id)
				acmejob.FailJob(client, job)
				return
			}
		} else {
			log.Errorf("[%s] [%d] No flight found for flight1 = `%d`", job.Type, jobKey, flight1.Id)
			acmejob.FailJob(client, job)
			return
		}
	}

	if flight2 != nil {
		endpoint := fmt.Sprintf("%s/flights/filter/", flight2.Airline)
		payload := map[string]interface{}{
			"code":              flight2.Code,
			"departure_airport": flight2.DepartureAirport,
			"departure_time":    flight2.DepartureTime,
			"arrival_airport":   flight2.ArrivalAirport,
			"arrival_time":      flight2.ArrivalTime,
		}

		response, err := http.MakeRequest(endpoint, payload)

		if err != nil {
			log.Errorf("[%s] [%d] Error for airline `%s`: %s", job.Type, jobKey, flight2.Airline, err.Error())
			acmejob.FailJob(client, job)
			return
		} else {
			if response.Count > 0 {
				if response.Count == 1 {
					flight2_id = int(response.Data[0]["id"].(float64))
				} else {
					log.Errorf("[%s] [%d] Found `%d` flights for flight2 = `%d`", job.Type, jobKey, response.Count, flight2.Id)
					acmejob.FailJob(client, job)
					return
				}
			} else {
				log.Errorf("[%s] [%d] No flight found for flight2 = `%d`", job.Type, jobKey, flight2.Id)
				acmejob.FailJob(client, job)
				return
			}
		}
	}

	conf, _ := config.GetConfig()

	endpoint = fmt.Sprintf("%s/login/", flight1.Airline)
	payload = map[string]interface{}{
		"username": conf.String("airline.login.username"),
		"password": conf.String("airline.login.password"),
	}

	token, err := http.MakeLogin(endpoint, payload)
	if err != nil {
		log.Errorf("[%s] [%d] Can't perform login: %s", job.Type, jobKey, err.Error())
		acmejob.FailJob(client, job)
		return
	}

	endpoint = fmt.Sprintf("%s/journeys/", flight1.Airline)
	payload = map[string]interface{}{
		"departure_flight_id": flight1_id,
		"cost":                offer.Journey.Cost,
		"email":               offer.User.Email,
	}

	if flight2_id != 0 {
		payload["arrival_flight_id"] = flight2_id
	}

	journeyResponse, err := http.NewJourneyRequest(endpoint, payload, *token)
	if err != nil {
		log.Errorf("[%s] [%d] Can't save new journey: %s", job.Type, jobKey, err.Error())
		acmejob.FailJob(client, job)
		return
	}

	variables["flight_price"] = offer.Journey.Cost
	log.Infof("[%s] [%d] Created a new new journey on airline company website with ID = %d", job.Type, jobKey, journeyResponse.Id)

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
