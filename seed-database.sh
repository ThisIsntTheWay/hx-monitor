#!/bin/bash
# Bootstraps the DB with an initial configuration
# Requires env vars to be set as expected by the go application itself; See README.md

echo "Attempting to connect to $MONGO_HOST:$MONGO_PORT..."
mongosh "mongodb://$MONGO_HOST:$MONGO_PORT" -u $MONGO_USER -p $MONGO_PASSWORD --authenticationDatabase admin --eval "$(cat <<EOF
conn = db.getMongo()
db = conn.getDB("hx");

db.numbers.insertMany([
    {
        _id: 1,
        description: "Meiringen",
        number: "0-800-496-347"
    }
])

EOF
)"

echo "> Have seeded DB"