package rmq

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CallTaskMessage represents a task to call an area
type CallTaskMessage struct {
	AreaID          primitive.ObjectID `json:"area_id"`
	AreaName        string             `json:"area_name"`
	NumberName      string             `json:"number_name"`
	PhoneNumber     string             `json:"phone_number"`
	RetryCount      int                `json:"retry_count"`
	MaxRetries      int                `json:"max_retries"`
	DoTranscription bool               `json:"do_transcription"`
	DoRecording     bool               `json:"do_recording"`
	Timestamp       time.Time          `json:"timestamp"`
}

// CallCompletedMessage represents a completed call (success or failure)
type CallCompletedMessage struct {
	AreaID        primitive.ObjectID `json:"area_id"`
	AreaName      string             `json:"area_name"`
	CallSID       string             `json:"call_sid"`
	Status        string             `json:"status"` // "success" or "failed"
	FailureReason string             `json:"failure_reason,omitempty"`
	RetryCount    int                `json:"retry_count"`
	MaxRetries    int                `json:"max_retries"`
	Timestamp     time.Time          `json:"timestamp"`
}

// PublishCallTask publishes a new call task to the queue
func PublishCallTask(msg CallTaskMessage) error {
	if _rmqConn == nil {
		return fmt.Errorf("RabbitMQ connection not initialized")
	}

	body, err := json.Marshal(msg)
	if err != nil {
		slog.Error("RMQ", "action", "PublishCallTask", "error", fmt.Sprintf("failed to marshal message: %v", err))
		return err
	}

	err = _rmqConn.channel.Publish(
		CallsExchange,
		CallsRoutingKey,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
		},
	)

	if err != nil {
		slog.Error("RMQ", "action", "PublishCallTask", "areaName", msg.AreaName, "error", err)
		return err
	}

	slog.Debug("RMQ", "action", "PublishCallTask", "areaName", msg.AreaName, "retryCount", msg.RetryCount)
	return nil
}

// PublishCallTaskDelayed publishes a call task to the delayed queue for retry
func PublishCallTaskDelayed(msg CallTaskMessage) error {
	if _rmqConn == nil {
		return fmt.Errorf("RabbitMQ connection not initialized")
	}

	body, err := json.Marshal(msg)
	if err != nil {
		slog.Error("RMQ", "action", "PublishCallTaskDelayed", "error", fmt.Sprintf("failed to marshal message: %v", err))
		return err
	}

	err = _rmqConn.channel.Publish(
		DelayedExchange,
		CallsDelayedRoutingKey,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
		},
	)

	if err != nil {
		slog.Error("RMQ", "action", "PublishCallTaskDelayed", "areaName", msg.AreaName, "error", err)
		return err
	}

	slog.Debug("RMQ", "action", "PublishCallTaskDelayed", "areaName", msg.AreaName, "retryCount", msg.RetryCount)
	return nil
}

// PublishCallCompleted publishes a call completion message
func PublishCallCompleted(msg CallCompletedMessage) error {
	if _rmqConn == nil {
		return fmt.Errorf("RabbitMQ connection not initialized")
	}

	body, err := json.Marshal(msg)
	if err != nil {
		slog.Error("RMQ", "action", "PublishCallCompleted", "error", fmt.Sprintf("failed to marshal message: %v", err))
		return err
	}

	err = _rmqConn.channel.Publish(
		CallsExchange,
		CallCompletedRoutingKey,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
		},
	)

	if err != nil {
		slog.Error("RMQ", "action", "PublishCallCompleted", "areaName", msg.AreaName, "error", err)
		return err
	}

	slog.Debug("RMQ", "action", "PublishCallCompleted", "areaName", msg.AreaName, "status", msg.Status)
	return nil
}

// ConsumeCallTasks returns a channel to consume call tasks
func ConsumeCallTasks() (<-chan amqp.Delivery, error) {
	if _rmqConn == nil {
		return nil, fmt.Errorf("RabbitMQ connection not initialized")
	}

	msgs, err := _rmqConn.channel.Consume(
		CallsQueueName,
		"",    // consumer
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)

	if err != nil {
		slog.Error("RMQ", "action", "ConsumeCallTasks", "error", err)
		return nil, err
	}

	return msgs, nil
}

// ConsumeCallCompleted returns a channel to consume call completion messages
func ConsumeCallCompleted() (<-chan amqp.Delivery, error) {
	if _rmqConn == nil {
		return nil, fmt.Errorf("RabbitMQ connection not initialized")
	}

	msgs, err := _rmqConn.channel.Consume(
		CallCompletedQueueName,
		"",    // consumer
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)

	if err != nil {
		slog.Error("RMQ", "action", "ConsumeCallCompleted", "error", err)
		return nil, err
	}

	return msgs, nil
}

// ConsumeDeadLetterCalls returns a channel to consume dead letter (max retries exceeded) messages
func ConsumeDeadLetterCalls() (<-chan amqp.Delivery, error) {
	if _rmqConn == nil {
		return nil, fmt.Errorf("RabbitMQ connection not initialized")
	}

	msgs, err := _rmqConn.channel.Consume(
		CallsDeadLetterQueue,
		"",    // consumer
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)

	if err != nil {
		slog.Error("RMQ", "action", "ConsumeDeadLetterCalls", "error", err)
		return nil, err
	}

	return msgs, nil
}
