# Setup
```bash
mongosh "mongodb://$MONGO_HOST:$MONGO_PORT" -u $MONGO_USER -p $MONGO_PASSWORD --authenticationDatabase admin

use hx
```

## Frontend
### Flight operating hours
#### 2 Times, now is within t1, t2
```bash
const currentTime = new Date();
const t1 = new Date(currentTime.getTime() - 2 * 60 * 60 * 1000);
const t2 = new Date(currentTime.getTime() + 2 * 60 * 60 * 1000);

db.hx_areas.updateOne(
    { name: "meiringen" },
    { $set: { "flight_operating_hours": [t1, t2] } }
)
```

Result: WITHIN flight operating hours

#### 2 Times, now is after t1 & t2
```bash
const currentTime = new Date();
const t1 = new Date(currentTime.getTime() - 4 * 60 * 60 * 1000);
const t2 = new Date(currentTime.getTime() - 2 * 60 * 60 * 1000);

db.hx_areas.updateOne(
    { name: "meiringen" },
    { $set: { "flight_operating_hours": [t1, t2] } }
)
```

Result: OUTSIDE flight operating hours

#### 4 Times, now is inbetween t2, t3
```bash
const currentTime = new Date();
const t1 = new Date(currentTime.getTime() - 4 * 60 * 60 * 1000);
const t2 = new Date(currentTime.getTime() - 2 * 60 * 60 * 1000);
const t3 = new Date(currentTime.getTime() + 2 * 60 * 60 * 1000);
const t4 = new Date(currentTime.getTime() + 4 * 60 * 60 * 1000);

db.hx_areas.updateOne(
    { name: "meiringen" },
    { $set: { "flight_operating_hours": [t1, t2, t3, t4] } }
)
```

Result: OUTSIDE flight operating hours

#### 4 Times, now is inbetween t3, t4
```bash
const currentTime = new Date();
const t1 = new Date(currentTime.getTime() - 6 * 60 * 60 * 1000);
const t2 = new Date(currentTime.getTime() - 4 * 60 * 60 * 1000);
const t3 = new Date(currentTime.getTime() - 2 * 60 * 60 * 1000);
const t4 = new Date(currentTime.getTime() + 4 * 60 * 60 * 1000);

db.hx_areas.updateOne(
    { name: "meiringen" },
    { $set: { "flight_operating_hours": [t1, t2, t3, t4] } }
)
```

Result: WITHIN flight operating hours