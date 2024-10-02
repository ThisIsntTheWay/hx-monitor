package main

import (
	"log/slog"
	"os"

	"github.com/thisisnttheway/hx-checker/callback"
	"github.com/thisisnttheway/hx-checker/caller"
	"github.com/thisisnttheway/hx-checker/db"
	"github.com/thisisnttheway/hx-checker/logger"
	"github.com/thisisnttheway/hx-checker/monitor"
)

// CHeck if certain env vars have been set
func preFlightChecks() {
	// Callback
	_, exists := os.LookupEnv("NGROK_AUTHTOKEN")
	if !exists {
		_, exists := os.LookupEnv("TWILIO_API_CALLBACK_URL")
		if !exists {
			logger.LogErrorFatal("MAIN", "Neither NGROK_AUTHTOKEN auth token or TWILIO_API_CALLBACK_URL is set")
		}
	}
}

func main() {
	preFlightChecks()

	db.Connect()

	slog.Info("MAIN", "message", "Attempting to get numbers...")
	numbers := caller.GetNumbers()
	for _, v := range numbers {
		slog.Info("MAIN", "number", v.Number, "name", v.Name)
	}

	// Callback URL handler
	go func() {
		callback.Serve()
	}()

	monitor.MonitorHxAreas()

	// Run ad infinitum
	select {}
}
