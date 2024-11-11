## Usage
Set env vars:
```bash
# MongoDB credentials
export MONGO_USER=
export MONGO_PASSWORD=
export MONGO_HOST=
export MONGO_PORT=27017

# Twilio
export TWILIO_REGION=ie1 # Unset to use us1
export TWILIO_ACCOUNT_SID=
export TWILIO_API_KEY=
export TWILIO_API_SECRET=

# Optional, shown are defaults
TWILIO_CALL_LENGTH=30 # In seconds
                      # English transcripts may take up to 38 seconds, e.g. Meiringen

TWILIO_CALLBACK_URL="" # If unset, will use ngrok to generate a callback URL
NGROK_AUTHTOKEN=""     # If TWILIO_CALLBACK_URL is unset, this must be set - see above
```

## Test
```bash
mkdir ./mongodb-test
export NGROK_AUTHTOKEN=abc

# Generate ngrok callback service BEFORE app is started
NGROK_CONTAINER_ID=$(docker ps -aq -f name=ngrok)
SYSTEM_IP=$(ip addr show eth0 | grep "inet\b" | awk '{print $2}' | cut -d/ -f1)
CALLBACK_SERVER_PORT=8080 #2343
if [ $(echo $NGROK_CONTAINER_ID | wc -l) -eq 0 ]; then
  docker run --name ngrok --net=host -d -e NGROK_AUTHTOKEN=$NGROK_AUTHTOKEN ngrok/ngrok:latest http http://$SYSTEM_IP:$CALLBACK_SERVER_PORT --log=stdout
else
  # Restart if stopped
  if [ $(docker ps -a -f name=ngrok | awk 'NR>1' | grep Exit | wc -l) -eq 1 ]; then
    docker start ngrok
  fi
fi

# Get ngrok URL
export TWILIO_CALLBACK_URL=$(docker logs ngrok | grep "url=" | cut -d "=" -f 8)
echo "TWILIO_CALLBACK_URL: $TWILIO_CALLBACK_URL"

# Local mongodb
docker run -d \
  --name mongo-local-test \
  -e MONGO_INITDB_ROOT_USERNAME=admin \
  -e MONGO_INITDB_ROOT_PASSWORD=adminpassword \
  -v ./mongodb-test:/data/db \
  -p 27017:27017 \
  mongodb-raspberrypi4-unofficial-r7.0.4 #mongo:latest
  # On RPi: https://github.com/themattman/mongodb-raspberrypi-docker

# Test mongodb
export MONGO_USER=admin
export MONGO_PASSWORD=adminpassword
export MONGO_HOST=localhost
export MONGO_PORT=27017

./seed-database.sh

# Twilio
export TWILIO_REGION=ie1
export TWILIO_ACCOUNT_SID=abc
export TWILIO_API_KEY=def
export TWILIO_API_SECRET=ghi
```