#!/bin/bash
# Bootstraps the DB with an initial configuration
# Requires env vars to be set as expected by the go application itself; See README.md

echo "Attempting to connect to $MONGO_HOST:$MONGO_PORT..."
mongosh "mongodb://$MONGO_HOST:$MONGO_PORT" -u $MONGO_USER -p $MONGO_PASSWORD --authenticationDatabase admin --eval "$(cat <<EOF
conn = db.getMongo()
db = conn.getDB("hx");

db.numbers.drop()
db.hx_areas.drop()

db.numbers.insertMany([
    {
        name: "meiringen",
        number: "0-800-496-347"
    }
])

db.hx_areas.insertMany([
    {
        full_name: "Meiringen TMA 1",
        area: "meiringen-tma-1",
        number_name: "meiringen"
    },
    {
        full_name: "Meiringen TMA 2",
        area: "meiringen-tma-2",
        number_name: "meiringen"
    },
    {
        full_name: "Meiringen TMA 3",
        area: "meiringen-tma-3",
        number_name: "meiringen"
    },
    {
        full_name: "Meiringen TMA 4",
        area: "meiringen-tma-4",
        number_name: "meiringen"
    },
    {
        full_name: "Meiringen TMA 5",
        area: "meiringen-tma-5",
        number_name: "meiringen"
    },
    {
        full_name: "Meiringen TMA 6",
        area: "meiringen-tma-6",
        number_name: "meiringen"
    },
    {
        full_name: "Meiringen CTX",
        area: "meiringen-ctx",
        number_name: "meiringen"
    },
])

EOF
)"

echo "> Have seeded DB"