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

# Program configuration
USE_TWILIO_TRANSCRIPTION=1  # bool, if set to true will instruct Twilio to transcribe with their STT
USE_WHISPER_TRANSCRIPTION=0 # bool, if set to true will use local whisper to transcribe
                            # Enabling this will both make and download recordings off Twilio
                            # Only one method of transcription may be used!                            

TWILIO_PARTIAL_TRANSCRIPTIONS=0 # bool, if set to true will instruct Twilio to send partial transcriptions
                                # Useful for scenarios where Twilio would only send a single transcribed sentence
                                # Will quickly result in HTTP 429 errors when using ngrok!

WHISPER_MODEL=tiny.en                 # Whisper model to use
WHISPER_MODELS_PATH=./models_whisper  # File path to whisper models
WHISPER_DO_MODEL_DOWNLOAD=1           # If WHISPER_MODEL was not found in ./models, attempt download from HuggingFace
                                      # Only supports models hosted in repository 'ggerganov/whisper.cpp'

TWILIO_CALL_LENGTH=30 # In seconds
                      # English transcripts may take up to 38 seconds, e.g. Meiringen

TWILIO_CALLBACK_URL="" # Publicly accessible (base) URL under which the callback server will be hosted
                       # If unset, will use ngrok to generate a callback URL
NGROK_AUTHTOKEN=""     # If TWILIO_CALLBACK_URL is unset, this must be set
```

## Test
```bash
mkdir ./mongodb-test

# Set either
export NGROK_AUTHTOKEN=abc
export TWILIO_CALLBACK_URL=abc

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

export USE_TWILIO_TRANSCRIPTION=1
export USE_WHISPER_TRANSCRIPTION=0
```