package main

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/thisisnttheway/hx-checker/configuration"
	"github.com/thisisnttheway/hx-checker/logger"
)

const port string = "8080"

func main() {
	configuration.SetUpMongoConfig()

	slog.Info("MAIN", "action", "startServer", "port", port, "apiBase", apiBase)
	err := http.ListenAndServe(":"+port, muxRouter)
	if err != nil {
		logger.LogErrorFatal("MAIN", fmt.Sprintf("Webserver was unable to start: %v", err))
	}
}
