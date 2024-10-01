```json
db={
  "numbers": [
    {
        "_id": 1,
        "description": "Interlaken",
        "number": "041 79 788 88 88",
        "last_called": "unix.Now()",
        "last_call_status": "reached|failed",
    }
  ],
  "hx_areas": [
    {
        "_id": 1,
        "full_name": "Interlaken TMA 1",
        "area": "interlaken-tma-1",
        "number_id": 1
    }
  ],
  "hx_status": [
    {
        "_id": 1,
        "status": "active|inactive|unknown",
        "date": "unix.Now()",
        "area_id": 1,
    }
  ],
  "calls": [
    {
        "_id": 1,
        "sid": "string",
        "time": "unix.Now()",
        "status": "good|fail",
        "status_verbose": "Number was reached",
        "cost": "0.01 USD",
        "number_id": 1
    }
  ],
  "transcripts": [
    {
        "_id": 1,
        "transcript": "My transcript",
        "date": "unix.Now()",
        "cost": "unknown",
        "number_id": 1,
        "hx_area_id": 1,
        "call_id": 1,
    }
  ]
}
```