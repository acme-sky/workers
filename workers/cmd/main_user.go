package main

import (
	"context"
	"fmt"
	"os"

	"github.com/acme-sky/bpmn/workers/internal/user"
	"github.com/camunda/zeebe/clients/go/v8/pkg/worker"
	"github.com/camunda/zeebe/clients/go/v8/pkg/zbc"
)

var jobStatuses = make(map[string](chan int))

type Job struct {
	name    string
	handler worker.JobHandler
}

func main() {
	ZeebeAddr := os.Getenv("ZEEBE_ADDRESS")
	BPMNFile := os.Getenv("BPMN_FILE")
	ProcessId := os.Getenv("PROCESS_ID")

	client, err := zbc.NewClient(&zbc.ClientConfig{
		GatewayAddress:         ZeebeAddr,
		UsePlaintextConnection: true,
	})

	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	// deploy process
	response, err := client.NewDeployResourceCommand().AddResourceFile(BPMNFile).Send(ctx)
	if err != nil {
		panic(err)
	}

	fmt.Println(response.String())

	result, err := client.NewCreateInstanceCommand().BPMNProcessId(ProcessId).LatestVersion().Send(ctx)
	if err != nil {
		panic(err)
	}

	fmt.Println(result.String())
	fmt.Println()

	jobs := []Job{
		{"TM_New_Request_Save_Flight", user.TMNewRequestSaveFlight},
	}

	for _, job := range jobs {
		client.NewJobWorker().JobType(job.name).Handler(job.handler).Open().AwaitClose()
	}
}
