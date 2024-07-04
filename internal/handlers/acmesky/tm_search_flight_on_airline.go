package handlers

import (
	"context"
	"fmt"

	"github.com/charmbracelet/log"

	"github.com/acme-sky/workers/internal/http"
	acmejob "github.com/acme-sky/workers/internal/job"
	"github.com/acme-sky/workers/internal/models"
	"github.com/camunda/zeebe/clients/go/v8/pkg/entities"
	"github.com/camunda/zeebe/clients/go/v8/pkg/worker"
)

// Task raised by ACMESky Flights manager lame in a sequential loop by "Get user
// interests".
// It makes a filter for airlines and set a variable `flight` is something is
// found. A request could be:
// curl -X POST <base>/flights/filter/ -H 'content-type: application/json' -H 'accept: application/json' \
// -d '{"departure_time":"2024-04-30T04:12:00+02:00","arrival_time":"2024-05-01T11:00:00+02:00","departure_airport":"CPH","arrival_airport":"CTA"}'
func TMSearchFlightsOnAirline(client worker.JobClient, job entities.Job) {
	jobKey := job.GetKey()

	variables, err := job.GetVariablesAsMap()
	if err != nil {
		acmejob.FailJob(client, job)
		return
	}

	airlines := variables["airlines"].([]interface{})
	index := int(variables["loopCounter"].(float64)) - 1
	airline := airlines[index].(models.Airline)

	interests := variables["interests"].([]interface{})

	if len(interests) == 0 {
		log.Warnf("Error for airline `%s`: there is no interest", airline.Name)
		acmejob.FailJob(client, job)
		return
	}

	if index < 0 || index >= len(interests) {
		log.Errorf("Error for airline `%s`: index out of range %d", airline.Name, index)
		acmejob.FailJob(client, job)
		return
	}

	flights := []map[string]interface{}{}
	endpoint := fmt.Sprintf("%s/flights/filter/", airline.Endpoint)
	for i := 0; i < len(interests); i++ {
		interest := interests[i].(map[string]interface{})
		interestId := int(interest["id"].(float64))
		user := interest["user"].(map[string]interface{})

		payload := map[string]interface{}{
			"departure_airport": interest["flight1_departure_airport"].(string),
			"departure_time":    interest["flight1_departure_time"].(string),
			"arrival_airport":   interest["flight1_arrival_airport"].(string),
			"arrival_time":      interest["flight1_arrival_time"].(string),
		}

		response, err := http.MakeRequest(endpoint, payload)

		if err != nil {
			log.Errorf("Error for airline `%s`: %s", airline.Name, err.Error())
		} else {
			if response.Count > 0 {
				for _, data := range response.Data {
					data["user_id"] = user["ID"]
					data["airline"] = airline.Name
					data["interest_id"] = interestId
					flights = append(flights, data)
				}
			}
		}

		if interest["flight2_departure_airport"] == nil {
			continue
		}

		payload = map[string]interface{}{
			"departure_airport": interest["flight2_departure_airport"].(string),
			"departure_time":    interest["flight2_departure_time"].(string),
			"arrival_airport":   interest["flight2_arrival_airport"].(string),
			"arrival_time":      interest["flight2_arrival_time"].(string),
		}

		response, err = http.MakeRequest(endpoint, payload)

		if err != nil {
			log.Errorf("Error for airline `%s`: %s", airline.Name, err.Error())
			continue
		}

		if response.Count > 0 {
			for _, data := range response.Data {
				data["user_id"] = user["ID"]
				data["airline"] = airline.Name
				data["interest_id"] = interestId
				flights = append(flights, data)
			}
		}
	}

	variables["flights"] = flights

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

	log.Infof("[%s] [%d] Successfully completed job with len(flights) = %d", job.Type, jobKey, len(flights))

	acmejob.JobVariables[job.Type] <- variables
	acmejob.JobStatuses.Close(job.Type, 0)
}
