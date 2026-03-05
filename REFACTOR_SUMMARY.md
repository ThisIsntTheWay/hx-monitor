# HX-Monitor Concurrency Refactor - Summary of Changes

## Overview
Successfully refactored the hx-monitor to use RabbitMQ for managing concurrent call processing instead of relying on database queries and in-memory state tracking.

## Files Created

### 1. `/monitor/rmq/rmq.go`
**Purpose**: RabbitMQ connection management and queue/exchange initialization

**Key Functions**:
- `Initialize(url string)`: Sets up RabbitMQ connection and declares all queues and exchanges
- `Get()`: Returns the initialized RabbitMQ connection
- `Close()`: Gracefully closes the connection

**Queue & Exchange Structure**:
- **Exchanges**: 
  - `hx_monitor_calls_exchange`: Main exchange for call tasks
  - `hx_monitor_delayed_exchange`: Delayed retry exchange  
  - `hx_monitor_dlx`: Dead letter exchange for failures

- **Queues**:
  - `hx_monitor_calls`: Main call task queue
  - `hx_monitor_calls_delayed`: 5-minute TTL delay queue with dead letter to main queue
  - `hx_monitor_calls_dlq`: Dead letter queue (max retries exceeded)
  - `hx_monitor_calls_completed`: Call completion notifications

### 2. `/monitor/rmq/messages.go`
**Purpose**: Message types and publisher/consumer functions

**Message Types**:
```go
type CallTaskMessage {
    AreaID, AreaName, NumberName, PhoneNumber string
    RetryCount, MaxRetries int
    DoTranscription, DoRecording bool
}

type CallCompletedMessage {
    AreaID, AreaName, CallSID string
    Status, FailureReason string
    RetryCount, MaxRetries int
}
```

**Key Functions**:
- `PublishCallTask(msg)`: Publishes immediate call task
- `PublishCallTaskDelayed(msg)`: Publishes to retry queue (5-min delay)
- `PublishCallCompleted(msg)`: Publishes call completion
- `ConsumeCallTasks()`: Returns channel for consuming tasks
- `ConsumeCallCompleted()`: Returns channel for completion messages
- `ConsumeDeadLetterCalls()`: Returns channel for dead letter messages

### 3. `/monitor/worker/worker.go`
**Purpose**: RabbitMQ message consumers and task processors

**Key Functions**:
- `StartCallTaskWorker()`: Consumes call tasks, invokes `ProcessCallTask()`, handles retries
- `StartCallCompletedConsumer()`: Consumes call completion messages
- `StartDeadLetterConsumer()`: Monitors failed tasks exceeding max retries

## Files Modified

### 1. `/monitor/monitor/monitor.go`
**Changes**:
- Removed: `areasNumberIsBeingCalled()` complex database aggregation query
- Removed: `_areaProcessingQueue` in-memory map
- Removed: `_areaFailureCounts` in-memory map  
- Removed: `onErrorNextActionDelay` (30-minute delay)
- Added: `MaxRetries` to `CallConfiguration` (default: 3)
- Added: `CheckAreaHasActiveCalls()` - lightweight DB check for active calls
- Updated: `initCall()` - now returns error on phone system not answering
  - Checks for: `no-answer`, `busy`, `failed` statuses
  - Provides detailed error messages
- Updated: `MonitorHxAreas()` 
  - Now publishes tasks to RabbitMQ instead of calling directly
  - Checks max retries and skips if exceeded
  - Updates `last_action` timestamp before publishing
- Added: `ProcessCallTask()` - new worker function to handle call execution
  - Handles success and failure cases
  - Manages retries (5-minute delay instead of 30)
  - Updates database with error count and status
  - Publishes `CallCompletedMessage` on success

### 2. `/monitor/main.go`
**Changes**:
- Added imports: `rmq` and `worker` packages
- Updated `init()`:
  - Initialize RabbitMQ connection with `rmq.Initialize()`
  - Reads `RABBITMQ_URL` environment variable (default: `amqp://guest:guest@localhost:5672/`)
- Updated `run()`:
  - Start three workers: `StartCallTaskWorker()`, `StartCallCompletedConsumer()`, `StartDeadLetterConsumer()`
  - Workers run in parallel goroutines

## Key Improvements

### 1. **Concurrency Model**
- **Before**: Blocking database queries to check active calls, in-memory state maps
- **After**: Asynchronous message-based queuing with persistent RabbitMQ messaging

### 2. **Retry Strategy**
- **Before**: 30-minute delays on failures
- **After**: 5-minute retries with configurable max attempts (default: 3)

### 3. **Phone System Handling**
- **Before**: Generic error handling, fatal logging
- **After**: Explicit detection of `no-answer`, `busy`, `failed` statuses with proper retry logic

### 4. **Scalability**
- **Before**: Single-threaded monitoring, synchronous calls
- **After**: Multiple workers can process tasks in parallel, messages persisted in RabbitMQ

### 5. **Error Tracking**
- **Before**: In-memory maps susceptible to loss on restart
- **After**: Dead letter queue tracks all failures for analysis and alerting

## Deployment Requirements

### Dependencies
```bash
go get github.com/rabbitmq/amqp091-go
```

### Environment Variables
- `RABBITMQ_URL`: RabbitMQ connection string (optional, defaults to localhost)
- `RABBITMQ_URL=amqp://user:pass@rabbitmq.example.com:5672/`

### Docker Compose Example
```yaml
rabbitmq:
  image: rabbitmq:3-management
  ports:
    - "5672:5672"
    - "15672:15672"
  environment:
    RABBITMQ_DEFAULT_USER: guest
    RABBITMQ_DEFAULT_PASS: guest
```

## Testing Checklist

- [ ] RabbitMQ connection initializes on startup
- [ ] Call tasks published to `hx_monitor_calls` queue
- [ ] Worker processes tasks and calls phone numbers
- [ ] Failed calls republished to delayed queue
- [ ] Retries occur after 5 minutes with correct retry count
- [ ] Max retries (3) exceeded sends to dead letter queue
- [ ] Phone system not answering detected and retried
- [ ] Successful calls reset error count to 0
- [ ] Call completion messages published correctly
- [ ] Dead letter consumer logs failures for alerting

## Migration Notes

### Database
No schema changes required. Existing `HXArea` fields are used:
- `num_errors`: Updated by worker (retry count)
- `last_action_success`: Boolean flag for success/failure
- `last_error`: Error description for debugging
- `last_action`: Timestamp when task was published

### API/Callbacks
The callback handler should be updated to publish `CallCompletedMessage` to RabbitMQ when calls complete. Currently the worker publishes on successful attempts.

### Monitoring
Monitor these RabbitMQ metrics:
- Queue depths (especially dead letter queue)
- Consumer count
- Message throughput
- Processing time per task

## Future Enhancements

1. **Horizontal Scaling**: Run multiple worker instances
2. **Priority Queues**: Different retry strategies for different area types
3. **Prometheus Metrics**: Export queue depth, processing times
4. **Circuit Breaker**: Auto-pause if phone system fails too often
5. **WebSocket Updates**: Real-time task status via WebSocket
6. **Task Prioritization**: Process critical areas first
