package handlers

import (
	"context"
	"fmt"
	"github.com/charmbracelet/log"
	"math/rand"

	acmejob "github.com/acme-sky/workers/internal/job"
	"github.com/camunda/zeebe/clients/go/v8/pkg/entities"
	"github.com/camunda/zeebe/clients/go/v8/pkg/worker"
)

func STRetrieveOffer(client worker.JobClient, job entities.Job) {
	jobKey := job.GetKey()

	variables, err := job.GetVariablesAsMap()
	if err != nil {
		acmejob.FailJob(client, job)
		return
	}
	variables["token_is_valid"] = true
	r := rand.Int()
	if r%2 == 0 {
		variables["token_is_valid"] = true
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
	acmejob.JobStatuses.Close(job.Type, 0)
}
