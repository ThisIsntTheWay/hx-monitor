# Must use parent dir as context
FROM golang:1.23.3-alpine AS build

WORKDIR /go/src/app/hx-monitor/monitor
COPY ./monitor .

WORKDIR /go/src/app/hx-monitor/api-callback
COPY ./api-backend .

RUN go clean; go mod tidy
RUN CGO_ENABLED=0 go build -o /go/bin/app

# -----------------------------------
FROM gcr.io/distroless/static-debian12

COPY --from=build /go/bin/app /
CMD ["/app"]