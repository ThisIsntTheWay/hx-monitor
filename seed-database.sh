#!/bin/bash
# Bootstraps the DB with an initial configuration
# Requires env vars to be set as expected by the go application itself; See README.md

echo "Attempting to connect to $MONGO_HOST:$MONGO_PORT..."
mongosh "mongodb://$MONGO_HOST:$MONGO_PORT" -u $MONGO_USER -p $MONGO_PASSWORD --authenticationDatabase admin --eval "$(cat <<EOF
conn = db.getMongo()
db = conn.getDB("hx");

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
        areas: [
            {
                full_name: "Meiringen CTR",
                name: "meiringen-ctr",
                status: false
            },
            {
                full_name: "Meiringen TMA 1",
                name: "meiringen-tma-1",
                status: false
            },
            {
                full_name: "Meiringen TMA 2",
                name: "meiringen-tma-2",
                status: false
            },
            {
                full_name: "Meiringen TMA 3",
                name: "meiringen-tma-3",
                status: false
            },
            {
                full_name: "Meiringen TMA 4",
                name: "meiringen-tma-4",
                status: false
            },
            {
                full_name: "Meiringen TMA 5",
                name: "meiringen-tma-5",
                status: false
            },
            {
                full_name: "Meiringen TMA 6",
                name: "meiringen-tma-6",
                status: false
            }
        ]
    },
])
EOF
)"

echo "> Have seeded DB"