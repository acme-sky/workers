package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/charmbracelet/log"

	"github.com/acme-sky/workers/internal/config"
	"github.com/acme-sky/workers/internal/http"
	acmejob "github.com/acme-sky/workers/internal/job"
	"github.com/acme-sky/workers/internal/models"
	"github.com/camunda/zeebe/clients/go/v8/pkg/entities"
	"github.com/camunda/zeebe/clients/go/v8/pkg/worker"
)

// Service used to save info into Prontogram backend service.
func STSaveInfoOnProntogram(client worker.JobClient, job entities.Job) {
	jobKey := job.GetKey()

	variables, err := job.GetVariablesAsMap()
	if err != nil {
		acmejob.FailJob(client, job)
		return
	}

	m := variables["offer"].(map[string]interface{})

	jsonData, err := json.Marshal(m)
	if err != nil {
		fmt.Println("Error marshaling map to JSON:", err)
		return
	}

	var offer models.Offer
	err = json.Unmarshal(jsonData, &offer)
	if err != nil {
		fmt.Println("Error unmarshaling JSON to struct:", err)
		return
	}

	conf, _ := config.GetConfig()
	endpoint := fmt.Sprintf("%s/sendMessage", conf.String("prontogram.endpoint"))

	fmt.Println(endpoint)
	expirationInt, _ := strconv.ParseInt(offer.Expired, 10, 64)
	expirationDate := time.Unix(expirationInt, 0)
	payload := http.ProntogramMessageRequest{
		Message:    offer.Message,
		Expiration: expirationDate.Format("2006-01-02T15:04:05Z"),
		Username:   *offer.User.ProntogramUsername,
		Sid:        " ",
	}
	_, err = http.MakeProntogramRequest(endpoint, payload)

	if err != nil {
		log.Errorf("[%s] [%d] Error for offer `%d`: %s", job.Type, jobKey, offer.Id, err.Error())
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

	acmejob.JobStatuses.Close(job.Type, 0)
}
