# RabbitMQ Integration - Usage Guide & Architecture

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                         MONITOR SCHEDULER                            │
│                      (Every 30 seconds)                              │
└──────────────────────────────┬──────────────────────────────────────┘
                               │
                               ▼
                    ┌──────────────────────┐
                    │ MonitorHxAreas()     │
                    │  - Check time        │
                    │  - Check active      │
                    │  - Publish task      │
                    └──────────────┬───────┘
                                   │
                ┌──────────────────┴──────────────────┐
                │                                     │
                ▼                                     ▼
        ┌─────────────────┐          ┌──────────────────────┐
        │ Skip (too early)│          │ RabbitMQ Exchange    │
        └─────────────────┘          │ (hx_calls_exchange)  │
                                     └──────────┬───────────┘
                                                │
                                       ┌────────┴─────────┐
                                       │                  │
                                       ▼                  ▼
                            ┌──────────────────┐  ┌──────────────┐
                            │  Main Queue      │  │  Completed   │
                            │(immediate calls) │  │   Queue      │
                            └────────┬─────────┘  └──────────────┘
                                     │
                                     ▼
          ┌──────────────────────────────────────────────────┐
          │      StartCallTaskWorker (goroutine)             │
          │         Consumes: hx_monitor_calls               │
          └──────────────────────┬───────────────────────────┘
                                 │
                    ┌────────────┴────────────┐
                    │                         │
                    ▼                         ▼
              ┌─────────────┐          ┌──────────────┐
              │ SUCCESS     │          │ FAILURE      │
              │             │          │              │
              │ emit:       │          │ Check retry  │
              │ completed   │          │ count        │
              │ reset       │          │              │
              │ errors: 0   │          └──────┬───────┘
              └─────────────┘                  │
                                   ┌──────────┴──────────┐
                                   │                     │
                ┌──────────────────▼─────────┐   ┌───────▼──────┐
                │ Retries < Max? (default 3) │   │ Max Retries  │
                │                            │   │  Exceeded    │
                │ YES: Delayed Queue         │   │              │
                │     (TTL: 5 minutes)       │   │ → DLQ (Alert)│
                └────────────────────────────┘   └──────────────┘
                        │
                   [Wait 5 min]
                        │
                        ▼
             ┌─────────────────────┐
             │ Requeue to Main     │
             │ (automatic via TTL) │
             └─────────────────────┘
```

## Workflow Examples

### Example 1: Successful Call

```
Time: 08:00:00
├─ Monitor checks: Area "Meiringen_CTR" is due for call
├─ Database lookup: num_errors = 0, within max retries
├─ Publish: CallTaskMessage to hx_monitor_calls
│  {
│    area_id: "ObjectId123",
│    area_name: "Meiringen_CTR",
│    phone_number: "+41123456789",
│    retry_count: 0,
│    max_retries: 3
│  }
│
├─ Worker receives message
├─ Calls phone number via Twilio
├─ Phone answers, call status = "in-progress"
├─ Publish: CallCompletedMessage
│  {
│    area_id: "ObjectId123",
│    status: "success",
│    call_sid: "CA123456789"
│  }
├─ Update DB:
│  └─ num_errors: 0
│  └─ last_action_success: true
│
└─ Message acknowledged (removed from queue)
```

### Example 2: Failed Call with Retry

```
Time: 08:00:00
├─ Monitor publishes call task (retry_count = 0)
│
├─ Worker receives message
├─ Calls phone - No answer (status: "no-answer")
├─ Error: "phone system not available: no-answer"
├─ Publish to delayed queue: CallTaskMessage
│  {
│    ...,
│    retry_count: 1,  ← incremented
│    max_retries: 3
│  }
├─ Update DB:
│  └─ num_errors: 1
│  └─ last_action_success: false
│  └─ last_error: "phone system not available: no-answer"
│
└─ Message TTL expires after 5 minutes
   
Time: 08:05:00
├─ Message automatically moved to main queue
│
├─ Worker receives message (retry_count = 1)
├─ Calls phone again
│  ... (success or failure with retry_count = 2)
```

### Example 3: Max Retries Exceeded

```
Time: 08:00:00  - First attempt (retry_count = 0) - Fails
Time: 08:05:00  - Second attempt (retry_count = 1) - Fails
Time: 08:10:00  - Third attempt (retry_count = 2) - Fails
Time: 08:15:00  - Would be fourth attempt...

At retry_count = 2:
├─ Worker detects retry_count >= max_retries
├─ Does NOT republish to delayed queue
├─ DLQ processing sends to dead letter queue
├─ Update DB: last_action_success = false, last_error = "Max retries exceeded"
│
├─ Dead Letter Consumer receives message
├─ Logs: 
│  "ERROR: Call task exceeded max retries"
│  "areaName: Meiringen_CTR"
│  "timeSincePublish: 15 minutes"
│
└─ Operator can see in logs/metrics that manual intervention needed
```

## Configuration

### Environment Variables

```bash
# RabbitMQ connection string
export RABBITMQ_URL=amqp://guest:guest@localhost:5672/

# Or with authentication
export RABBITMQ_URL=amqp://myuser:mypass@rabbitmq.example.com:5672/
```

### Call Configuration (in code)

```go
// Modify in monitor/monitor/monitor.go init()

_callConfiguration.DoTranscription = true      // Enable transcription
_callConfiguration.DoRecording = false         // Disable recording
_callConfiguration.MaxRetries = 3              // Max retry attempts after initial failure

// 5-minute delay is hardcoded in RabbitMQ queue TTL
// To change: modify rmq/rmq.go queue declaration
// "x-message-ttl": int32(5 * 60 * 1000)  ← milliseconds
```

## Monitoring & Debugging

### Check Queue Status

```bash
# Via RabbitMQ Management UI
curl -u guest:guest http://localhost:15672/api/queues/% | jq .

# Check specific queue depth
curl -u guest:guest http://localhost:15672/api/queues/%2F/hx_monitor_calls | jq '.messages'
```

### View Logs

```bash
# Monitor publishing tasks
grep "PublishCallTask" logs.txt

# Check worker processing
grep "ProcessCallTask" logs.txt

# See dead letter events
grep "ProcessDeadLetter" logs.txt

# Follow all RabbitMQ activity
grep "action.*RMQ\|WORKER" logs.txt
```

### Database Queries

```javascript
// Check current error state
db.hx_areas.find({ 
  num_errors: { $gt: 0 } 
}, { 
  name: 1, 
  num_errors: 1, 
  last_error: 1,
  last_action_success: 1
})

// Check areas with max retries exceeded
db.hx_areas.find({ 
  num_errors: { $gte: 3 } 
})

// Recent failures in call collection
db.calls.find({ 
  status: { $in: ["failed", "no-answer", "busy"] },
  time: { $gte: new Date(Date.now() - 3600000) }  // Last hour
}).sort({ time: -1 })
```

## Scaling & Performance

### Single Instance (Current)
- Monitor: 1 scheduler process checking every 30 seconds
- Worker: 1 goroutine consuming from main queue
- Processing: Sequential, ~5 calls per 5 minutes = 1 call/minute max

### Horizontal Scaling

```go
// Start multiple workers in worker/worker.go

// Run in separate container/process:
for i := 0; i < 5; i++ {  // 5 parallel workers
  go worker.StartCallTaskWorker()
}

// Now processing: 5 tasks in parallel
// Throughput: ~5 calls/minute
```

### Performance Tuning

1. **Increase `sleepTime` in monitor/main.go**
   - Default: 30 seconds
   - If too many false positives: increase to 60 seconds
   - If too slow response: decrease to 15 seconds

2. **Adjust `MaxRetries`**
   - Default: 3
   - For critical areas: increase to 5
   - For non-critical: decrease to 2

3. **Adjust TTL delay**
   - Default: 5 minutes (300 seconds)
   - Faster retries: decrease to 2 minutes
   - Slower retries: increase to 10 minutes

4. **Database indexes**
   - Ensure indexes on `calls.number_id`, `hx_areas.number_name`
   - Improves CheckAreaHasActiveCalls() query speed

## Troubleshooting

### Issue: Tasks not being processed
- Check RabbitMQ is running: `docker ps | grep rabbitmq`
- Check connection: `grep "Initialize" logs | grep -i "error"`
- Verify RABBITMQ_URL setting

### Issue: Messages stuck in queue
- Check for JSON parsing errors: `grep "failed to unmarshal" logs`
- Message format might have changed
- Consider dead lettering manually: Management UI

### Issue: Too many retries
- Database might be slow: optimize CheckAreaHasActiveCalls()
- Phone system overloaded: increase TTL delay
- Network issues: check Twilio/network connectivity

### Issue: Dead letter queue growing
- Phone system consistently unavailable
- Check phone number configuration
- Consider reducing MaxRetries for that area
