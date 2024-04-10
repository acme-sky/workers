package main

import (
	"context"
	"fmt"
	"os"

	acmejob "github.com/acme-sky/bpmn/workers/internal/job"
	"github.com/acme-sky/bpmn/workers/internal/user"
	"github.com/camunda/zeebe/clients/go/v8/pkg/worker"
	"github.com/camunda/zeebe/clients/go/v8/pkg/zbc"
)

type HandlerType int

const (
	JobType HandlerType = iota
	MessageType
)

type Job struct {
	name           string
	handler        worker.JobHandler
	_type          HandlerType
	correlationKey string
}

func main() {
	ZeebeAddr := os.Getenv("ZEEBE_ADDRESS")
	BPMNFile := os.Getenv("BPMN_FILE")
	ProcessId := os.Getenv("PROCESS_ID")

	client, err := zbc.NewClient(&zbc.ClientConfig{
		GatewayAddress:         ZeebeAddr,
		UsePlaintextConnection: true,
	})

	defer client.Close()

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
		{"TM_New_Request_Save_Flight", user.TMNewRequestSaveFlight, JobType, ""},
		{"RM_Ack_Flight_Request_Save", nil, MessageType, "0"},
	}

	for _, job := range jobs {
		switch job._type {
		case JobType:
			acmejob.JobStatuses[job.name] = make(chan int, 1)

			client.NewJobWorker().JobType(job.name).Handler(job.handler).Open()
			<-acmejob.JobStatuses[job.name]

			defer client.Close()
			break
		case MessageType:
			res, err := client.NewPublishMessageCommand().MessageName(job.name).CorrelationKey(job.correlationKey).Send(ctx)
			if err != nil {
				println("err", err)
			} else {
				println("res", res.String())
			}
			break
		}
	}
}
