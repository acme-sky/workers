package job

import (
	"context"
	"github.com/charmbracelet/log"
	"os"
	"sync"

	"github.com/camunda/zeebe/clients/go/v8/pkg/commands"
	"github.com/camunda/zeebe/clients/go/v8/pkg/entities"
	"github.com/camunda/zeebe/clients/go/v8/pkg/pb"
	"github.com/camunda/zeebe/clients/go/v8/pkg/worker"
	"github.com/camunda/zeebe/clients/go/v8/pkg/zbc"
)

type jobStatusesMap struct {
	mu sync.Mutex
	m  map[string](chan int)
}

var JobStatuses = jobStatusesMap{m: make(map[string](chan int))}
var JobVariables = make(map[string](chan map[string]interface{}))
var JobAfter = make(map[string](chan int))

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

type PostJobHandler func(*zbc.Client, context.Context)

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

	var err error
	var client zbc.Client

	if client, err = zbc.NewClient(&zbc.ClientConfig{
		GatewayAddress:         ZeebeAddr,
		UsePlaintextConnection: true,
	}); err != nil {
		panic(err)
	}

	ctx := context.Background()

	var response *pb.DeployResourceResponse

	if response, err = client.NewDeployResourceCommand().AddResourceFile(BPMNFile).Send(ctx); err != nil {
		panic(err)
	}

	log.Infof(response.String())

	variables := map[string]interface{}{"airlines": []int{1, 2, 3}}

	var instance commands.CreateInstanceCommandStep3
	if instance, err = client.NewCreateInstanceCommand().BPMNProcessId(ProcessId).LatestVersion().VariablesFromMap(variables); err != nil {
		panic(err)
	}

	var result *pb.CreateProcessInstanceResponse

	if result, err = instance.Send(ctx); err != nil {
		panic(err)
	}

	log.Infof(result.String())

	return &client
}

func HandleJob(client *zbc.Client, job Job) {
	ctx := context.Background()

	ch := make(chan int, 1)
	JobStatuses.Set(job.Name, ch)

	JobVariables[job.Name] = make(chan map[string]interface{}, 1)
	JobAfter[job.Name] = make(chan int, 1)

	// TODO: study why multi-instance jobs does not fit this close-worker below
	// worker := (*client).NewJobWorker().JobType(job.Name).Handler(job.Handler).Open()
	(*client).NewJobWorker().JobType(job.Name).Handler(job.Handler).Open()

	if job.Message != nil {
		var variables map[string]interface{}
		ok := true
		select {
		case variables, ok = <-JobVariables[job.Name]:
			if !ok {
				log.Errorf("Channel JobVariables for %s is already closed\n", job.Name)
				panic("Reuse of closed channel")
			}
		}
		res, err := (*client).NewPublishMessageCommand().MessageName(job.Message.Name).CorrelationKey(job.Message.CorrelationKey).VariablesFromMap(variables)

		if err != nil {
			log.Error(err.Error())
		} else {
			if _, err := res.Send(ctx); err != nil {
				log.Error(err.Error())
			} else {
				log.Infof("Sent message to `%s` with correlation key = `%s`\n", job.Message.Name, job.Message.CorrelationKey)
			}
		}
	}

	if job.After != nil {
		select {
		case _, ok := <-JobAfter[job.Name]:
			if !ok {
				log.Errorf("Channel JobAfter for %s is already closed\n", job.Name)
				panic("Reuse of closed channel")
			} else {
				job.After(client, ctx)
			}
		}
	}
	// worker.Close()
	// worker.AwaitClose()

	JobStatuses.Get(job.Name)
	<-ch

	// HandleJob(client, job)
}

func FailJob(client worker.JobClient, job entities.Job) {
	log.Error("Failed to complete job", "job", job.GetKey())

	ctx := context.Background()
	_, err := client.NewFailJobCommand().JobKey(job.GetKey()).Retries(job.Retries - 1).Send(ctx)
	if err != nil {
		panic(err)
	}
}
