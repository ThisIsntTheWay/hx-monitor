package worker

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/thisisnttheway/hx-monitor/monitor"
	"github.com/thisisnttheway/hx-monitor/rmq"
)

// StartCallTaskWorker starts the worker that processes call tasks from RabbitMQ
// This should be run in a goroutine in main.go
func StartCallTaskWorker() error {
	slog.Info("WORKER", "action", "StartCallTaskWorker", "status", "starting")

	msgs, err := rmq.ConsumeCallTasks()
	if err != nil {
		slog.Error("WORKER", "action", "StartCallTaskWorker", "error", err)
		return err
	}

	// Process messages in a goroutine
	go func() {
		for msg := range msgs {
			slog.Debug("WORKER", "action", "ProcessCallTask", "bodySize", len(msg.Body))

			// Parse the message
			var task rmq.CallTaskMessage
			if err := json.Unmarshal(msg.Body, &task); err != nil {
				slog.Error("WORKER",
					"action", "ProcessCallTask",
					"error", fmt.Sprintf("failed to unmarshal message: %v", err),
				)
				// Nack the message to requeue it
				msg.Nack(false, true)
				continue
			}

			// Process the call task
			if err := monitor.ProcessCallTask(task); err != nil {
				slog.Warn("WORKER",
					"action", "ProcessCallTask",
					"areaName", task.AreaName,
					"error", err,
				)
			}

			// Acknowledge the message after processing
			msg.Ack(false)
		}
		slog.Info("WORKER", "action", "StartCallTaskWorker", "status", "stopped")
	}()

	return nil
}

// StartCallCompletedConsumer starts the consumer that listens for call completion messages
// This should be run in a goroutine in main.go or in the callback module
func StartCallCompletedConsumer() error {
	slog.Info("WORKER", "action", "StartCallCompletedConsumer", "status", "starting")

	msgs, err := rmq.ConsumeCallCompleted()
	if err != nil {
		slog.Error("WORKER", "action", "StartCallCompletedConsumer", "error", err)
		return err
	}

	go func() {
		for msg := range msgs {
			slog.Debug("WORKER", "action", "ProcessCallCompleted", "bodySize", len(msg.Body))

			var completed rmq.CallCompletedMessage
			if err := json.Unmarshal(msg.Body, &completed); err != nil {
				slog.Error("WORKER",
					"action", "ProcessCallCompleted",
					"error", fmt.Sprintf("failed to unmarshal message: %v", err),
				)
				msg.Nack(false, true)
				continue
			}

			slog.Info("WORKER",
				"action", "ProcessCallCompleted",
				"areaName", completed.AreaName,
				"status", completed.Status,
				"callSID", completed.CallSID,
			)

			// Acknowledge the message
			msg.Ack(false)
		}
		slog.Info("WORKER", "action", "StartCallCompletedConsumer", "status", "stopped")
	}()

	return nil
}

// StartDeadLetterConsumer starts the consumer for messages that exceeded max retries
// This should be run in a goroutine in main.go
func StartDeadLetterConsumer() error {
	slog.Info("WORKER", "action", "StartDeadLetterConsumer", "status", "starting")

	msgs, err := rmq.ConsumeDeadLetterCalls()
	if err != nil {
		slog.Error("WORKER", "action", "StartDeadLetterConsumer", "error", err)
		return err
	}

	go func() {
		for msg := range msgs {
			slog.Debug("WORKER", "action", "ProcessDeadLetter", "bodySize", len(msg.Body))

			var task rmq.CallTaskMessage
			if err := json.Unmarshal(msg.Body, &task); err != nil {
				slog.Error("WORKER",
					"action", "ProcessDeadLetter",
					"error", fmt.Sprintf("failed to unmarshal message: %v", err),
				)
				msg.Nack(false, false)
				continue
			}

			slog.Error("WORKER",
				"action", "ProcessDeadLetter",
				"areaName", task.AreaName,
				"message", "Call task exceeded max retries",
				"retryCount", task.RetryCount,
				"maxRetries", task.MaxRetries,
				"timestamp", task.Timestamp,
				"timeSincePublish", time.Since(task.Timestamp).String(),
			)

			// Here you could send alerts, emails, etc.
			// For now, just log the dead letter

			msg.Ack(false)
		}
		slog.Info("WORKER", "action", "StartDeadLetterConsumer", "status", "stopped")
	}()

	return nil
}
