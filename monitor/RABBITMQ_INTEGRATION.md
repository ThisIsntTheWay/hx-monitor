# RabbitMQ Integration - Concurrency Improvements

## Overview

This refactor improves the concurrency model of the hx-monitor by replacing database-based processing state tracking with RabbitMQ message queuing. This eliminates expensive database queries and enables true decoupled processing of call tasks.

## Key Changes

### 1. **Removed Database-Dependent Concurrency Checks**

**Before:**
- `areasNumberIsBeingCalled()` performed complex MongoDB aggregation queries to check if an area had active calls
- Used in-memory `_areaProcessingQueue` and `_areaFailureCounts` maps for tracking state
- Synchronous call execution within the monitor loop

**After:**
- Replaced with lightweight `CheckAreaHasActiveCalls()` that only queries the call collection for non-completed calls
- RabbitMQ handles concurrency via message queuing
- Asynchronous task processing through dedicated workers

### 2. **RabbitMQ Architecture**

A new `rmq` package handles all RabbitMQ operations:

**Queues:**
- `hx_monitor_calls`: Main queue for immediate call tasks
- `hx_monitor_calls_delayed`: Temporary queue with 5-minute TTL for retries (auto-requeued to main queue)
- `hx_monitor_calls_dlq`: Dead letter queue for messages that exceeded max retries
- `hx_monitor_calls_completed`: For call completion notifications

**Exchanges:**
- `hx_monitor_calls_exchange`: Direct exchange for call tasks and completion messages
- `hx_monitor_delayed_exchange`: Direct exchange for delayed retry routing
- `hx_monitor_dlx`: Dead letter exchange for failed tasks

### 3. **Improved Retry Logic**

**5-Minute Retry Strategy:**
- Failed calls are republished to the delayed queue instead of 30-minute delays
- Delayed messages automatically expire and requeue to the main queue after 5 minutes
- MaxRetries configurable (default: 3)
- Exceeding max retries sends messages to dead letter queue for alerting

**Phone System Not Answering:**
- `initCall()` now checks call status for: `no-answer`, `busy`, `failed`
- These conditions are treated as failures and trigger retries
- Error messages logged with specific failure reasons

### 4. **Worker Pattern**

New `worker` package provides three consumers:

1. **StartCallTaskWorker()**: Processes call tasks from the main queue
   - Calls `monitor.ProcessCallTask()`
   - Handles retries by republishing to delayed queue
   - Manages error counts in the database

2. **StartCallCompletedConsumer()**: Listens for call completion messages
   - Currently just logs completion
   - Can be extended to trigger callbacks or webhooks

3. **StartDeadLetterConsumer()**: Handles messages exceeding max retries
   - Logs dead letter events for alerting
   - Can be extended to send notifications

## Integration Steps

### 1. Add RabbitMQ Dependency

```bash
go get github.com/rabbitmq/amqp091-go
```

### 2. Update main.go

The refactored `main.go` includes:

```go
// Initialize RabbitMQ in init()
rabbitmqURL := os.Getenv("RABBITMQ_URL")
if rabbitmqURL == "" {
    rabbitmqURL = "amqp://guest:guest@localhost:5672/"
}
if err := rmq.Initialize(rabbitmqURL); err != nil {
    logger.LogErrorFatal("MAIN", "Failed to initialize RabbitMQ: "+err.Error())
}

// Start workers in run()
if err := worker.StartCallTaskWorker(); err != nil {
    return err
}
if err := worker.StartCallCompletedConsumer(); err != nil {
    return err
}
if err := worker.StartDeadLetterConsumer(); err != nil {
    return err
}
```

### 3. Environment Variables

Required:
- `RABBITMQ_URL`: RabbitMQ connection string (default: `amqp://guest:guest@localhost:5672/`)

### 4. Database Schema

No schema changes required. The system still uses the same HXArea model with:
- `num_errors`: Retry count (increased by workers)
- `last_action_success`: Boolean flag
- `last_error`: Error message from last failure

## Message Structures

### CallTaskMessage
```json
{
  "area_id": "ObjectId",
  "area_name": "string",
  "number_name": "string",
  "phone_number": "string",
  "retry_count": int,
  "max_retries": int,
  "do_transcription": bool,
  "do_recording": bool,
  "timestamp": "time"
}
```

### CallCompletedMessage
```json
{
  "area_id": "ObjectId",
  "area_name": "string",
  "call_sid": "string",
  "status": "success|failed",
  "failure_reason": "string",
  "retry_count": int,
  "max_retries": int,
  "timestamp": "time"
}
```

## Flow Diagram

```
MonitorHxAreas() [scheduler]
    ↓
Checks if time to call
    ↓
Publishes CallTaskMessage to RabbitMQ
    ↓
[5-minute gap...]
    ↓
StartCallTaskWorker [consumer]
    ↓
Consumes message from main queue
    ↓
ProcessCallTask()
    ├─ Success → PublishCallCompleted() → Updates DB (num_errors = 0)
    └─ Failure → PublishCallTaskDelayed() → Updates DB (num_errors++)
              If retry_count >= max_retries → Goes to DLQ
    ↓
StartDeadLetterConsumer [optional alerting]
    ↓
Log/Alert on max retries exceeded
```

## Benefits

1. **Reduced Database Load**: Eliminates expensive aggregation queries for checking active calls
2. **Decoupled Processing**: Call scheduling and execution happen independently
3. **Fault Tolerant**: RabbitMQ persists messages across restarts
4. **Scalable**: Multiple workers can be run in separate processes/containers
5. **Faster Retries**: 5-minute retry delays instead of 30 minutes
6. **Better Error Handling**: Phone system failures explicitly detected and handled

## Testing

### Local RabbitMQ Setup
```bash
docker run -d --name rabbitmq -p 5672:15672 rabbitmq:3-management
# Access management UI at http://localhost:15672 (guest:guest)
```

### Verify Queue Creation
```bash
# After running the monitor
curl http://localhost:15672/api/queues/% -u guest:guest
```

## Future Enhancements

1. **Multiple Worker Instances**: Scale horizontally by running multiple worker instances
2. **Priority Queues**: Add priority levels for urgent vs. routine checks
3. **Metrics/Observability**: Export Prometheus metrics for queue depth, processing times
4. **Callback Integration**: Have callback handler publish `CallCompletedMessage` directly
5. **Circuit Breaker**: Auto-pause calls if phone system repeatedly unavailable
