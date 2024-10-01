## Test
```bash
mkdir ./mongodb-test
export NGROK_AUTHTOKEN=abc

NGROK_CONTAINER_ID=$(docker ps -aq -f name=ngrok)
if [ $(echo $NGROK_CONTAINER_ID | wc -l) -eq 0 ]; then
  docker run --name ngrok --net=host -d -e NGROK_AUTHTOKEN=$NGROK_AUTHTOKEN ngrok/ngrok:latest http 8080 --log=stdout
else
  # Restart
  if [ $(docker ps -a -f name=ngrok | awk 'NR>1' | grep Exit | wc -l) -eq 1 ]; then
    docker start ngrok
  fi
fi

# Get ngrok URL
export TWILIO_CALLBACK_URL=$(docker logs ngrok | grep "url=" | cut -d "=" -f 8)
echo "TWILIO_CALLBACK_URL: $TWILIO_CALLBACK_URL"

# On RPi: https://github.com/themattman/mongodb-raspberrypi-docker
docker run -d \
  --name mongo-local-test \
  -e MONGO_INITDB_ROOT_USERNAME=admin \
  -e MONGO_INITDB_ROOT_PASSWORD=adminpassword \
  -v ./mongodb-test:/data/db \
  -p 27017:27017 \
  mongodb-raspberrypi4-unofficial-r7.0.4 #mongo:latest

# Test mongodb
export MONGO_USER=admin
export MONGO_PASSWORD=adminpassword
export MONGO_HOST=localhost
export MONGO_PORT=27017

# Twilio
export TWILIO_REGION=ie1
export TWILIO_ACCOUNT_SID=abc
export TWILIO_API_KEY=def
export TWILIO_API_SECRET=ghi
```