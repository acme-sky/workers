package acme

import (
	"context"
	"fmt"
	"log"
	"math/rand"

	acmejob "github.com/acme-sky/bpmn/workers/internal/job"
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
		variables["token_is_valid"] = false
	}

	request, err := client.NewCompleteJobCommand().JobKey(jobKey).VariablesFromMap(variables)
	if err != nil {
		acmejob.FailJob(client, job)
		return
	}

	log.SetPrefix(fmt.Sprintf("[%s] [%d] ", job.Type, jobKey))
	log.Println("Job completed")
	log.Println("Processing data:", variables)

	ctx := context.Background()
	_, err = request.Send(ctx)
	if err != nil {
		acmejob.FailJob(client, job)
		panic(err)
	}

	log.Println("Successfully completed job")
	close(acmejob.JobStatuses[job.Type])
}
