#!/bin/bash
# Bootstraps the DB with an initial configuration
# Requires env vars to be set as expected by the go application itself; See README.md

echo "Attempting to connect to $MONGO_HOST:$MONGO_PORT..."
mongosh "mongodb://$MONGO_HOST:$MONGO_PORT" -u $MONGO_USER -p $MONGO_PASSWORD --authenticationDatabase admin --eval "$(cat <<EOF
conn = db.getMongo()
db = conn.getDB("hx");

db.calls.drop()
db.numbers.drop()
db.hx_areas.drop()
db.hx_sub_areas.drop()

db.numbers.insertMany([
    {
        name: "meiringen",
        number: "+41800496347"
    }
])

db.hx_areas.insertMany([
    {
        name: "meiringen",
        number_name: "meiringen",
        sub_areas: [
            {
                full_name: "CTR Meiringen HX",
                name: "meiringen-ctr",
                active: false
            },
            {
                full_name: "TMA Meiringen 1 HX",
                name: "meiringen-tma-1",
                active: false
            },
            {
                full_name: "TMA Meiringen 2 HX",
                name: "meiringen-tma-2",
                active: false
            },
            {
                full_name: "TMA Meiringen 3 HX",
                name: "meiringen-tma-3",
                active: false
            },
            {
                full_name: "TMA Meiringen 4 HX",
                name: "meiringen-tma-4",
                active: false
            },
            {
                full_name: "TMA Meiringen 5 HX",
                name: "meiringen-tma-5",
                active: false
            },
            {
                full_name: "TMA Meiringen 6 HX",
                name: "meiringen-tma-6",
                active: false
            }
        ]
    },
])
EOF
)"

echo "> Have seeded DB"