package message

import (
	"context"
	"encoding/json"
	"os"

	"github.com/camunda/zeebe/clients/go/v8/pkg/zbc"
	"github.com/charmbracelet/log"
	amqp "github.com/rabbitmq/amqp091-go"
)

// Body received in message to perform a new job
type MessageBody struct {
	// Create a new message request for this name
	Name string `json:"name"`

	// Corellation key value
	CorrelationKey string `json:"correlation_key"`

	// Json payload value
	Payload map[string]interface{} `json:"payload"`
}

// Instance a RabbitMQ message broker for messaging management
func MessageBroker(client *zbc.Client) {
	log.SetPrefix("[RabbitMQ]")
	ctx := context.Background()
	conn, err := amqp.Dial(os.Getenv("RABBITMQ_URI"))

	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %s", err.Error())
	} else {
		log.Info("Connected to RabbitMQ")
	}

	defer conn.Close()

	ch, err := conn.Channel()

	if err != nil {
		log.Fatalf("Failed to open a channel: %s", err.Error())
	}

	defer ch.Close()

	q, err := ch.QueueDeclare("acme_messages", false, true, false, false, nil)

	if err != nil {
		log.Fatalf("Failed to declare a queue: %s", err.Error())
	}

	if err := ch.Qos(1, 0, false); err != nil {
		log.Fatalf("Failed to set QoS: %s", err.Error())
	}

	msgs, err := ch.Consume(q.Name, "", false, false, false, false, nil)

	if err != nil {
		log.Fatalf("Failed to register a consumer: %s", err.Error())
	}

	var forever chan struct{}

	for d := range msgs {
		var body MessageBody
		if err := json.Unmarshal(d.Body, &body); err != nil {
			log.Errorf("Error on a received message: %s", err.Error())
			continue
		}
		res, err := (*client).NewPublishMessageCommand().MessageName(body.Name).CorrelationKey(body.CorrelationKey).VariablesFromMap(body.Payload)

		if err != nil {
			log.Error(err.Error())
		} else {
			if _, err := res.Send(ctx); err != nil {
				log.Error(err.Error())
			} else {
				log.Infof("Sent message to `%s` with correlation key = `%s` with payload = `%v`\n", body.Name, body.CorrelationKey, body.Payload)
			}
		}
		d.Ack(false)
	}

	<-forever
}
