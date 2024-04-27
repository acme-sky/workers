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

// A safer map using a mutex system to avoid concurrent writes
type jobStatusesMap struct {
	mu sync.Mutex
	m  map[string](chan int64)
}

// Map used to sync jobs
var JobStatuses = jobStatusesMap{m: make(map[string](chan int64))}

// Map used to sync variables for jobs
var JobVariables = make(map[string](chan map[string]interface{}))

// Map used to sync after function handler for jobs
var JobAfter = make(map[string](chan int))

// Set function for a `key` in the map
func (sm *jobStatusesMap) Set(key string, value chan int64) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.m[key] = value
}

// This function should closes the channel but, since we have an issue here,
// just edit the value.
// FIXME: should close the channel `close(sm.m[key])`
func (sm *jobStatusesMap) Close(key string, value int64) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.m[key] <- value
}

// Get the value for the map with a `key`
func (sm *jobStatusesMap) Get(key string) (chan int64, bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	value, ok := sm.m[key]
	return value, ok
}

// Struct used for the publish message command executed by the client.
type MessageCommand struct {
	// Name of the BPMN' message catch event
	Name string

	// Correlation key of the message catch event. Actually, we do not really
	// use this field properly
	CorrelationKey string
}

// Works as the same as JobHandler but it is called after the handler execution
type AfterJobHandler func(*zbc.Client, context.Context)

// The Job structure used by all the BPMN activities
type Job struct {
	// Name of the task
	Name string

	// Handler function
	Handler worker.JobHandler

	// A possible message to send after the response of the hanlder
	Message *MessageCommand

	// A possibile function to be executed after the response of the handler
	After AfterJobHandler
}

// Handle the job instance for the `client`
func (job *Job) Handle(client *zbc.Client) {
	ctx := context.Background()

	// Start all the channel used to sync status, variables and after function
	ch := make(chan int64, 1)
	JobStatuses.Set(job.Name, ch)

	JobVariables[job.Name] = make(chan map[string]interface{}, 1)
	JobAfter[job.Name] = make(chan int, 1)

	// TODO: study why multi-instance jobs does not fit this close-worker below
	// worker := (*client).NewJobWorker().JobType(job.Name).Handler(job.Handler).Open()
	(*client).NewJobWorker().JobType(job.Name).Handler(job.Handler).Open()

	if job.Message != nil {
		// It waites until `JobVariables[job.Name]` returns a value. Then it
		// publishes the message
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
		// It waites until `JobAfter[job.Name]` returns a value. Then it
		// publishes the message
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

	value, _ := JobStatuses.Get(job.Name)
	pid := <-value
	ch <- pid
	if pid != 0 {
		if _, err := (*client).NewCancelInstanceCommand().ProcessInstanceKey(pid).Send(ctx); err != nil {
			log.Errorf("Error canceling the instance: %s", err.Error())
		}
	}

	job.Handle(client)
}

// Job used in case of a failure. Create a new `FailJobCommand` and retry. In
// case of error on the "retry", just panic the routine.
func FailJob(client worker.JobClient, job entities.Job) {
	log.Error("Failed to complete job", "job", job.GetKey())

	ctx := context.Background()
	_, err := client.NewFailJobCommand().JobKey(job.GetKey()).Retries(0).Send(ctx)
	if err != nil {
		log.Errorf("Error %s", err.Error())
	}

	JobStatuses.Close(job.Type, job.ProcessInstanceKey)
}

// Main function whcih creates a new Zeebe client.
// If called with the parameter `pid`, that value will be run as `ProcessId`
func CreateClient(pid string) *zbc.Client {
	// Load some variables from the environment
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

	// Airlines must be loaded for the first time as variables 'cause the timer
	// trigger executed every hour.
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
