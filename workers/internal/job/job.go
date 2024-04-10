package job

import (
	"context"
	"fmt"
	"github.com/camunda/zeebe/clients/go/v8/pkg/entities"
	"github.com/camunda/zeebe/clients/go/v8/pkg/worker"
	"github.com/camunda/zeebe/clients/go/v8/pkg/zbc"
	"log"
	"os"
)

var JobStatuses = make(map[string](chan int))

type HandlerType int

const (
	JobType HandlerType = iota
	MessageType
)

type Job struct {
	Name           string
	Handler        worker.JobHandler
	Type_          HandlerType
	CorrelationKey string
}

func CreateClient(pid string) *zbc.Client {
	ZeebeAddr := os.Getenv("ZEEBE_ADDRESS")
	BPMNFile := os.Getenv("BPMN_FILE")
	ProcessId := os.Getenv("PROCESS_ID")
	if len(pid) != 0 {
		ProcessId = pid
	}

	client, err := zbc.NewClient(&zbc.ClientConfig{
		GatewayAddress:         ZeebeAddr,
		UsePlaintextConnection: true,
	})

	if err != nil {
		panic(err)
	}

	ctx := context.Background()

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

	return &client
}

func HandleJobs(client *zbc.Client, jobs []Job) {
	ctx := context.Background()

	for _, job := range jobs {
		switch job.Type_ {
		case JobType:
			JobStatuses[job.Name] = make(chan int, 1)

			worker := (*client).NewJobWorker().JobType(job.Name).Handler(job.Handler).Open()
			<-JobStatuses[job.Name]

			worker.Close()
			worker.AwaitClose()

			break
		case MessageType:
			res, err := (*client).NewPublishMessageCommand().MessageName(job.Name).CorrelationKey(job.CorrelationKey).Send(ctx)
			if err != nil {
				log.Printf("[%s] %s\n", job.Name, err.Error())
			} else {
				log.Printf("[%s] %s\n", job.Name, res.String())
			}
			break
		}
	}
}

func FailJob(client worker.JobClient, job entities.Job) {
	log.Println("Failed to complete job", job.GetKey())

	ctx := context.Background()
	_, err := client.NewFailJobCommand().JobKey(job.GetKey()).Retries(job.Retries - 1).Send(ctx)
	if err != nil {
		panic(err)
	}
}
