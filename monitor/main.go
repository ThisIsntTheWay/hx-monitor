package main

import (
	"flag"
	"log/slog"
	"os"
	"time"

	"github.com/thisisnttheway/hx-monitor/callback"
	"github.com/thisisnttheway/hx-monitor/caller"
	"github.com/thisisnttheway/hx-monitor/configuration"
	"github.com/thisisnttheway/hx-monitor/db"
	"github.com/thisisnttheway/hx-monitor/logger"
	"github.com/thisisnttheway/hx-monitor/monitor"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	sleepTime time.Duration = 30 * time.Second
	forceCall *bool
)

// Check if certain env vars have been set
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

// Returns the nearest NextAction time of hx_areas. Default: time.Now()
func getNearestNextActionTime() time.Time {
	result := time.Now()
	type AggregateResult struct {
		NextAction time.Time `bson:"next_action"`
	}

	// Only get next_action and sort ascending
	results, err := db.Aggregate[AggregateResult]("hx_areas", mongo.Pipeline{
		bson.D{{"$sort", bson.D{
			{"next_action", 1},
		}}},
		bson.D{{"$project", bson.D{
			{"_id", false},
			{"next_action", true},
		}}},
	})

	if err == nil && len(results) > 0 {
		result = results[0].NextAction
	} else {
		slog.Warn("MAIN",
			"action", "getNearestNextActionTime",
			"message", "Using default value instead of DB",
			"returnValue", result,
			"errorDb", err,
		)
	}

	return result
}

func init() {
	preFlightChecks()
	db.Connect()

	// Callback URL handler
	go func() {
		callback.Serve()
	}()
}

func run() error {
	// Set up config
	slog.Debug("MAIN", "event", "setUpTwilioConfig")
	configuration.SetUpTwilioConfig()

	slog.Debug("MAIN", "event", "getNumbers")
	numbers := caller.GetNumbers()
	for _, v := range numbers {
		slog.Info("MAIN",
			"action", "indexNumbers",
			"number", v.Number,
			"name", v.Name,
		)
	}

	// Main loop
	for {
		nextActionableTime := getNearestNextActionTime()
		if *forceCall || time.Now().After(nextActionableTime) {
			*forceCall = false
			slog.Info("MAIN",
				"action", "monitorHxAreas",
			)

			monitor.MonitorHxAreas()
		} else {
			slog.Info("MAIN", "action", "awaitNextAction",
				"eta", time.Until(nextActionableTime),
				"nextActionTime", nextActionableTime,
			)
		}

		time.Sleep(sleepTime)
	}
}

func main() {
	forceCall = flag.Bool("force-call", false, "Force immediate processing of areas")
	flag.Parse()

	err := run()
	if err != nil {
		slog.Error("MAIN", "error", err)
		os.Exit(1)
	}
}
