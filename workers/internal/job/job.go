package job

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/camunda/zeebe/clients/go/v8/pkg/entities"
	"github.com/camunda/zeebe/clients/go/v8/pkg/worker"
	"github.com/camunda/zeebe/clients/go/v8/pkg/zbc"
)

var JobStatuses = make(map[string](chan int))
var JobVariables = make(map[string](chan string))

type MessageCommand struct {
	Name           string
	CorrelationKey string
}

type Job struct {
	Name    string
	Handler worker.JobHandler
	Message *MessageCommand
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

func HandleJob(client *zbc.Client, job Job) {
	ctx := context.Background()

	JobStatuses[job.Name] = make(chan int, 1)
	JobVariables[job.Name] = make(chan string, 1)

	worker := (*client).NewJobWorker().JobType(job.Name).Handler(job.Handler).Open()

	if job.Message != nil {
		variables := <-JobVariables[job.Name]
		res, err := (*client).NewPublishMessageCommand().MessageName(job.Message.Name).CorrelationKey(job.Message.CorrelationKey).VariablesFromString(variables)

		if err != nil {
			log.Println(err.Error())
		} else {
			if _, err := res.Send(ctx); err != nil {
				log.Println(err.Error())
			} else {
				log.Printf("Sent message to `%s` with correlation key = `%s`\n", job.Message.Name, job.Message.CorrelationKey)
			}
		}
	}

	<-JobStatuses[job.Name]
	worker.Close()
	worker.AwaitClose()

	time.Sleep(10 * time.Second)
	HandleJob(client, job)
}

func FailJob(client worker.JobClient, job entities.Job) {
	log.Println("Failed to complete job", job.GetKey())

	ctx := context.Background()
	_, err := client.NewFailJobCommand().JobKey(job.GetKey()).Retries(job.Retries - 1).Send(ctx)
	if err != nil {
		panic(err)
	}
}
