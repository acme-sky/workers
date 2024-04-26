package handlers

import (
	"context"
	"fmt"
	"github.com/charmbracelet/log"
	"math/rand"

	acmejob "github.com/acme-sky/workers/internal/job"
	"github.com/camunda/zeebe/clients/go/v8/pkg/entities"
	"github.com/camunda/zeebe/clients/go/v8/pkg/worker"
	"github.com/camunda/zeebe/clients/go/v8/pkg/zbc"
)

func TMAskPaymentLink(client worker.JobClient, job entities.Job) {
	jobKey := job.GetKey()

	variables, err := job.GetVariablesAsMap()
	if err != nil {
		acmejob.FailJob(client, job)
		return
	}

	request, err := client.NewCompleteJobCommand().JobKey(jobKey).VariablesFromMap(variables)
	if err != nil {
		acmejob.FailJob(client, job)
		return
	}

	log.SetPrefix(fmt.Sprintf("[%s] [%d] ", job.Type, jobKey))

	log.Debug("Processing data:", variables)

	ctx := context.Background()
	_, err = request.Send(ctx)
	if err != nil {
		acmejob.FailJob(client, job)
		return
	}

	log.Infof("Successfully completed job")
	acmejob.JobVariables[job.Type] <- variables
	acmejob.JobAfter[job.Type] <- 0

	acmejob.JobStatuses.Close(job.Type)
}

// Simulate a response from Bank participant
func TMAskPaymentLinkAfter(client *zbc.Client, ctx context.Context) {
	variables := map[string]interface{}{"payment_status": "ERR"}

	if rand.Int()%2 == 0 {
		variables["payment_status"] = "ERR"
	}

	res, err := (*client).NewPublishMessageCommand().MessageName("CM_Payment_Response").CorrelationKey("0").VariablesFromMap(variables)

	if err != nil {
		log.Infof(err.Error())
	} else {
		if _, err := res.Send(ctx); err != nil {
			log.Infof(err.Error())
		} else {
			log.Infof("Sent message to `CM_Payment_Response` with correlation key = `0` and %v", variables)
		}
	}
}
