package message

import (
	"os"

	"github.com/charmbracelet/log"
	amqp "github.com/rabbitmq/amqp091-go"
)

// Instance a RabbitMQ message broker for messaging management
func MessageBroker() {
	log.SetPrefix("[RabbitMQ]")
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
		log.Infof("Received a message: %s", d.Body)
		d.Ack(false)
	}

	<-forever
}
