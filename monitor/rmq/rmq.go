package rmq

import (
	"fmt"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	// Queue names
	CallsQueueName         = "hx_monitor_calls"
	CallsDelayedQueueName  = "hx_monitor_calls_delayed"
	CallsDeadLetterQueue   = "hx_monitor_calls_dlq"
	CallCompletedQueueName = "hx_monitor_calls_completed"

	// Exchange names
	CallsExchange      = "hx_monitor_calls_exchange"
	DelayedExchange    = "hx_monitor_delayed_exchange"
	DeadLetterExchange = "hx_monitor_dlx"

	// Routing keys
	CallsRoutingKey         = "calls.new"
	CallsDelayedRoutingKey  = "calls.delayed"
	CallCompletedRoutingKey = "calls.completed"
)

type RabbitMQConnection struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

var _rmqConn *RabbitMQConnection

// Initialize RabbitMQ connection and declare queues/exchanges
func Initialize(url string) error {
	slog.Info("RMQ", "action", "initialize", "url", url)

	conn, err := amqp.Dial(url)
	if err != nil {
		slog.Error("RMQ", "action", "initialize", "error", fmt.Sprintf("failed to connect: %v", err))
		return err
	}

	channel, err := conn.Channel()
	if err != nil {
		slog.Error("RMQ", "action", "initialize", "error", fmt.Sprintf("failed to open channel: %v", err))
		return err
	}

	_rmqConn = &RabbitMQConnection{
		conn:    conn,
		channel: channel,
	}

	// Declare exchanges
	if err := declareExchanges(channel); err != nil {
		return err
	}

	// Declare queues
	if err := declareQueues(channel); err != nil {
		return err
	}

	slog.Info("RMQ", "action", "initialize", "status", "success")
	return nil
}

func declareExchanges(ch *amqp.Channel) error {
	// Main exchange for call tasks
	if err := ch.ExchangeDeclare(
		CallsExchange,
		"direct",
		true,  // durable
		false, // auto-delete
		false, // internal
		false, // no-wait
		nil,   // args
	); err != nil {
		slog.Error("RMQ", "action", "declareExchanges", "exchange", CallsExchange, "error", err)
		return err
	}

	// Delayed exchange for retry logic
	if err := ch.ExchangeDeclare(
		DelayedExchange,
		"direct",
		true,  // durable
		false, // auto-delete
		false, // internal
		false, // no-wait
		nil,   // args
	); err != nil {
		slog.Error("RMQ", "action", "declareExchanges", "exchange", DelayedExchange, "error", err)
		return err
	}

	// Dead letter exchange for final failures
	if err := ch.ExchangeDeclare(
		DeadLetterExchange,
		"direct",
		true,  // durable
		false, // auto-delete
		false, // internal
		false, // no-wait
		nil,   // args
	); err != nil {
		slog.Error("RMQ", "action", "declareExchanges", "exchange", DeadLetterExchange, "error", err)
		return err
	}

	return nil
}

func declareQueues(ch *amqp.Channel) error {
	// Main calls queue
	if _, err := ch.QueueDeclare(
		CallsQueueName,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		amqp.Table{
			"x-dead-letter-exchange":    DeadLetterExchange,
			"x-dead-letter-routing-key": "calls.dlq",
		},
	); err != nil {
		slog.Error("RMQ", "action", "declareQueues", "queue", CallsQueueName, "error", err)
		return err
	}

	// Delayed calls queue (for retries with expiration)
	if _, err := ch.QueueDeclare(
		CallsDelayedQueueName,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		amqp.Table{
			"x-dead-letter-exchange":    CallsExchange,
			"x-dead-letter-routing-key": CallsRoutingKey,
			"x-message-ttl":             int32(5 * 60 * 1000), // 5 minutes in milliseconds
		},
	); err != nil {
		slog.Error("RMQ", "action", "declareQueues", "queue", CallsDelayedQueueName, "error", err)
		return err
	}

	// Dead letter queue for messages that exceeded max retries
	if _, err := ch.QueueDeclare(
		CallsDeadLetterQueue,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,
	); err != nil {
		slog.Error("RMQ", "action", "declareQueues", "queue", CallsDeadLetterQueue, "error", err)
		return err
	}

	// Completed calls queue (for callbacks)
	if _, err := ch.QueueDeclare(
		CallCompletedQueueName,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,
	); err != nil {
		slog.Error("RMQ", "action", "declareQueues", "queue", CallCompletedQueueName, "error", err)
		return err
	}

	// Bind queues to exchanges
	exchanges := []struct {
		queue      string
		exchange   string
		routingKey string
	}{
		{CallsQueueName, CallsExchange, CallsRoutingKey},
		{CallsDelayedQueueName, DelayedExchange, CallsDelayedRoutingKey},
		{CallsDeadLetterQueue, DeadLetterExchange, "calls.dlq"},
		{CallCompletedQueueName, CallsExchange, CallCompletedRoutingKey},
	}

	for _, binding := range exchanges {
		if err := ch.QueueBind(
			binding.queue,
			binding.routingKey,
			binding.exchange,
			false,
			nil,
		); err != nil {
			slog.Error("RMQ", "action", "declareQueues", "action", "QueueBind",
				"queue", binding.queue, "exchange", binding.exchange, "error", err)
			return err
		}
	}

	return nil
}

// Get returns the initialized RabbitMQ connection
func Get() *RabbitMQConnection {
	return _rmqConn
}

// GetChannel returns the AMQP channel
func (rc *RabbitMQConnection) GetChannel() *amqp.Channel {
	return rc.channel
}

// GetConnection returns the AMQP connection
func (rc *RabbitMQConnection) GetConnection() *amqp.Connection {
	return rc.conn
}

// Close closes the RabbitMQ connection
func (rc *RabbitMQConnection) Close() error {
	if rc.channel != nil {
		if err := rc.channel.Close(); err != nil {
			slog.Error("RMQ", "action", "close", "resource", "channel", "error", err)
		}
	}
	if rc.conn != nil {
		if err := rc.conn.Close(); err != nil {
			slog.Error("RMQ", "action", "close", "resource", "connection", "error", err)
			return err
		}
	}
	return nil
}
