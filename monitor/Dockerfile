FROM golang:1.23.3-alpine AS build

ENV CGO_ENABLED=0
RUN apk add --no-cache git make g++

COPY . /app
WORKDIR /app

RUN /bin/sh /app/build.sh

# ----------------------
FROM gcr.io/distroless/static-debian12

COPY --from=build /app/hx-monitor /
COPY --from=mwader/static-ffmpeg:7.1 /ffmpeg /usr/local/bin/
COPY --from=mwader/static-ffmpeg:7.1 /ffprobe /usr/local/bin/

CMD ["/hx-monitor"]