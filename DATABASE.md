```json
db={
  "numbers": [
    {
      "_id": 1,
      "name": "meiringen",
      "number": "041 79 788 88 88",
      "last_called": "time.Time",
      "last_call_status": "reached|failed",
    }
  ],
  "hx_areas": [
    {
      "_id": 1,
      "name": "meiringen",
      "next_action": "time.Time",
      "last_action": "time.Time",
      "number_name": "meiringen"
    }
  ],
  "hx_sub_areas": [
    {
      "_id": 1,
      "full_name": "Meiringen TMA 1",
      "name": "meiringen-tma-1",
      "status": "active|inactive|unknown",
      "area_reference": "meiringen"
    }
  ],
  "calls": [
    {
      "_id": 1,
      "sid": "string",
      "time": "time.Time",
      "status": "good|fail",
      "cost": "0.01 USD",
      "number_id": 1
    }
  ],
  "transcripts": [
    {
        "_id": 1,
        "transcript": "My transcript",
        "date": "time.Time",
        "cost": "unknown",
        "number_id": 1,
        "hx_area_id": 1,
        "call_id": 1,
    }
  ]
}
```