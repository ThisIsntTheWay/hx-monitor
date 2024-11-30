FROM golang:1.23.3-alpine

# ToDo: compile whisper through another build stage
RUN apk add --no-cache git make g++

COPY --from=mwader/static-ffmpeg:7.1 /ffmpeg /usr/local/bin/
COPY --from=mwader/static-ffmpeg:7.1 /ffprobe /usr/local/bin/

COPY . /app

WORKDIR /app
RUN /bin/sh /app/build.sh