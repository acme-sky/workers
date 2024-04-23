package handlers

import (
	"context"
	"fmt"
	"github.com/charmbracelet/log"

	acmejob "github.com/acme-sky/bpmn/workers/internal/job"
	"github.com/camunda/zeebe/clients/go/v8/pkg/entities"
	"github.com/camunda/zeebe/clients/go/v8/pkg/worker"
)

func TMSendOffer(client worker.JobClient, job entities.Job) {
	jobKey := job.GetKey()

	variables, err := job.GetVariablesAsMap()
	if err != nil {
		acmejob.FailJob(client, job)
		return
	}

	payload := variables
	payload["message"] = "Hello John Doe, this is the offer token for your flight from <b>BLQ</b> to <b>CPH</b> in date April 10th 11:10 - April 10th 13:30.<br><a href=\"#\" target=\"_blank\">1234</a>"
	payload["expired"] = "1712855681"
	payload["user"] = "sa"

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
	acmejob.JobVariables[job.Type] <- payload

	acmejob.JobStatuses.Close(job.Type)
}
