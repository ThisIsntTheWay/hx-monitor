FROM golang:1.23.3-alpine AS build

RUN apk add --no-cache git make g++

COPY . /app
WORKDIR /app
RUN /bin/sh /app/build.sh

# Required for whisper to work
COPY --from=mwader/static-ffmpeg:7.1 /ffmpeg /usr/local/bin/
COPY --from=mwader/static-ffmpeg:7.1 /ffprobe /usr/local/bin/

CMD ["/app/hx-monitor"] 