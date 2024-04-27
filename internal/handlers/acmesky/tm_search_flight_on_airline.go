package handlers

import (
	"context"
	"fmt"

	"github.com/charmbracelet/log"

	"github.com/acme-sky/workers/internal/http"
	acmejob "github.com/acme-sky/workers/internal/job"
	"github.com/camunda/zeebe/clients/go/v8/pkg/entities"
	"github.com/camunda/zeebe/clients/go/v8/pkg/worker"
)

// Task raised by ACMESky Flights manager lame in a sequential loop by "Get user
// interests".
// It makes a filter for airlines and set a variable `flight` is something is
// found. A request could be:
// curl -X POST <base>/flights/filter/ -H 'content-type: application/json' -H 'accept: application/json' \
// -d '{"departaure_time":"2024-04-30T04:12:00+02:00","arrival_time":"2024-05-01T11:00:00+02:00","departaure_airport":"CPH","arrival_airport":"CTA"}'
func TMSearchFlightsOnAirline(client worker.JobClient, job entities.Job) {
	jobKey := job.GetKey()

	variables, err := job.GetVariablesAsMap()
	if err != nil {
		acmejob.FailJob(client, job)
		return
	}

	airlines := variables["airlines"].([]interface{})
	index := int(variables["loopCounter"].(float64)) - 1
	airline := airlines[index]

	interests := variables["interests"].([]interface{})
	if index < 0 || index >= len(interests) {
		panic("Index out of range")
	}

	flights := []map[string]interface{}{}
	endpoint := fmt.Sprintf("%s/flights/filter/", airline)
	for i := 0; i < len(interests); i++ {
		interest := interests[i].(map[string]interface{})
		user := interest["user"].(map[string]interface{})

		payload := map[string]interface{}{
			"departaure_airport": interest["flight1_departaure_airport"].(string),
			"departaure_time":    interest["flight1_departaure_time"].(string),
			"arrival_airport":    interest["flight1_arrival_airport"].(string),
			"arrival_time":       interest["flight1_arrival_time"].(string),
		}

		response, err := http.MakeRequest(endpoint, payload)

		if err != nil {
			log.Errorf("Error for airline `%s`: %s", airline, err.Error())
		} else {
			if response.Count > 0 {
				for _, data := range response.Data {
					data["user_id"] = user["ID"]
					flights = append(flights, data)
				}
			}
		}

		if interest["flight2_departaure_airport"] == nil {
			continue
		}

		payload = map[string]interface{}{
			"departaure_airport": interest["flight2_departaure_airport"].(string),
			"departaure_time":    interest["flight2_departaure_time"].(string),
			"arrival_airport":    interest["flight2_arrival_airport"].(string),
			"arrival_time":       interest["flight2_arrival_time"].(string),
		}

		response, err = http.MakeRequest(endpoint, payload)

		if err != nil {
			log.Errorf("Error for airline `%s`: %s", airline, err.Error())
			continue
		}

		if response.Count > 0 {
			for _, data := range response.Data {
				data["user_id"] = user["ID"]
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
