package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/thisisnttheway/hx-checker/configuration"
	"github.com/thisisnttheway/hx-checker/db"
	"github.com/thisisnttheway/hx-checker/logger"
)

const defaultPort string = "8080"

func init() {
	db.Connect()
}

func main() {
	configuration.SetUpMongoConfig()

	listenPort, exists := os.LookupEnv("LISTEN_PORT")
	if !exists {
		slog.Warn("MAIN", "message", "LISTEN_PORT is unset, using default", "default", defaultPort)
		listenPort = defaultPort
	}

	slog.Info("MAIN", "action", "startServer", "port", listenPort, "apiBase", apiBase)
	err := http.ListenAndServe(":"+listenPort, muxRouter)
	if err != nil {
		logger.LogErrorFatal("MAIN", fmt.Sprintf("Webserver was unable to start: %v", err))
	}
}
