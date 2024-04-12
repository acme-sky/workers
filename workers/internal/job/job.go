package job

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/camunda/zeebe/clients/go/v8/pkg/entities"
	"github.com/camunda/zeebe/clients/go/v8/pkg/worker"
	"github.com/camunda/zeebe/clients/go/v8/pkg/zbc"
)

type jobStatusesMap struct {
	mu sync.Mutex
	m  map[string](chan int)
}

var JobStatuses = jobStatusesMap{m: make(map[string](chan int))}
var JobVariables = make(map[string](chan map[string]interface{}))
func (sm *jobStatusesMap) Set(key string, value chan int) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.m[key] = value
}

func (sm *jobStatusesMap) Close(key string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.m[key] <- 0
}

func (sm *jobStatusesMap) Get(key string) (chan int, bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	value, ok := sm.m[key]
	return value, ok
}

type MessageCommand struct {
	Name           string
	CorrelationKey string
}

type PostJobHandler func(*zbc.Client)

type Job struct {
	Name    string
	Handler worker.JobHandler
	Message *MessageCommand
	After   PostJobHandler
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

	ch := make(chan int, 1)
	JobStatuses.Set(job.Name, ch)

	JobVariables[job.Name] = make(chan map[string]interface{}, 1)

	// TODO: study why multi-instance jobs does not fit this close-worker below
	// worker := (*client).NewJobWorker().JobType(job.Name).Handler(job.Handler).Open()
	(*client).NewJobWorker().JobType(job.Name).Handler(job.Handler).Open()

	if job.Message != nil {
		var variables map[string]interface{}
		ok := true
		select {
		case variables, ok = <-JobVariables[job.Name]:
			if !ok {
				log.Panicf("Channel JobVariables for %s is already closed\n", job.Name)
			}
		}
		res, err := (*client).NewPublishMessageCommand().MessageName(job.Message.Name).CorrelationKey(job.Message.CorrelationKey).VariablesFromMap(variables)

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

	// worker.Close()
	// worker.AwaitClose()

	println("--------------\n", job.Name, "\n-------------_")
	JobStatuses.Get(job.Name)
	<-ch

	// close(ch)
	// HandleJob(client, job)
}

func FailJob(client worker.JobClient, job entities.Job) {
	log.Println("Failed to complete job", job.GetKey())

	ctx := context.Background()
	_, err := client.NewFailJobCommand().JobKey(job.GetKey()).Retries(job.Retries - 1).Send(ctx)
	if err != nil {
		panic(err)
	}
}
