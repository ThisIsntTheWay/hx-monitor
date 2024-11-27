package monitor

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/thisisnttheway/hx-checker/caller"
	"github.com/thisisnttheway/hx-checker/db"
	"github.com/thisisnttheway/hx-checker/logger"
	"github.com/thisisnttheway/hx-checker/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type ActionableNumber struct {
	NumberName string
	MustActNow bool
}

// { "<area>": <num_fails> }
var _areaFailureCounts map[string]int8
var maxFailsPerArea int8 = 3

// Determines if an area is being processed based on its last_action timestamp and associated, non-completed calls
func areaIsBeingProcessed(area models.HXArea) (bool, error) {
	// based on models.HXArea
	type AggregateResult struct {
		AreaID      primitive.ObjectID `bson:"_id"`
		AreaName    string             `bson:"name"`
		CallDetails []models.Call      `bson:"call_details"`
	}

	results, err := db.Aggregate[AggregateResult]("hx_areas", mongo.Pipeline{
		bson.D{{"$match", bson.M{"_id": area.ID}}},

		// Enumerate numbers and calls
		bson.D{{"$lookup", bson.D{
			{"from", "numbers"},
			{"localField", "number_name"},
			{"foreignField", "name"},
			{"as", "number_details"},
		}}},
		bson.D{{"$unwind", "$number_details"}},
		bson.D{{"$lookup", bson.D{
			{"from", "calls"},
			{"localField", "number_details._id"},
			{"foreignField", "number_id"},
			{"as", "call_details"},
		}}},

		// Only return select fields and further filter call_details
		bson.D{{"$project", bson.D{
			{"_id", true},
			{"name", true},
			{"last_action", true},
			{"call_details", bson.D{
				{"$filter", bson.D{
					{"input", "$call_details"},
					{"cond", bson.D{
						{"$gte", bson.A{"$$this.time", area.LastAction}},
					}},
				}}},
			}},
		}},
	})
	if err != nil {
		slog.Error("MONITOR",
			"action", "aggregateHxAreas",
			"error", err,
			"areaId", area.ID,
			"areaLastAction", area.LastAction,
		)
		return false, err
	}

	if len(results) == 0 {
		slog.Error("MONITOR",
			"action", "aggregateHxAreas",
			"error", "Length of aggregation result is 0",
			"areaId", area.ID,
			"areaLastAction", area.LastAction,
		)
	}

	hasCompletedCalls := false
	if len(results[0].CallDetails) > 0 {
		for _, s := range results[0].CallDetails {
			if s.Status == "completed" {
				hasCompletedCalls = true
				break
			}
		}
	} else {
		slog.Warn("MONITOR",
			"action", "aggregateHxAreas",
			"message", "Area has had no calls older than referenceTime",
			"areaId", area.ID,
			"areaName", area.Name,
			"referenceTime", area.LastAction,
		)
	}

	o, _ := json.Marshal(results)
	slog.Debug("MONITOR",
		"action", "aggregateHxAreas",
		"hasCompletedCalls", hasCompletedCalls,
		"resultFromDb", string(o),
	)

	// If area has completed calls = Area is not being processed
	return !hasCompletedCalls, nil
}

// Increments the amount of fails for an area and returns the amount of fails (post increment)
func incrementAreaFails(areaName string) int8 {
	v, ok := _areaFailureCounts[areaName]
	if ok {
		_areaFailureCounts[areaName] = v + 1
	} else {
		_areaFailureCounts[areaName] = 1
	}

	return _areaFailureCounts[areaName]
}

// Removes area failures for a given area
func removeAreaFails(areaName string) {
	_, exists := _areaFailureCounts[areaName]
	if exists {
		delete(_areaFailureCounts, areaName)
	}
}

// Call a number and start transcription
func initCallAndTranscription(number string) caller.CallResponse {
	call, err := caller.Call(number, true)
	if err != nil {
		slog.Error("MONITOR",
			"message", fmt.Sprintf("Failure calling number '%s'", number),
			"error", err,
		)

		logger.LogErrorFatal("MONITOR", err.Error())
	}

	return call
}

// Monitor HX areas: Keep track of states and schedule calls if necessary
func MonitorHxAreas() {
	hxAreas, err := db.GetDocument[models.HXArea]("hx_areas", bson.D{})
	if len(hxAreas) == 0 || err != nil {
		logger.LogErrorFatal("MONITOR",
			fmt.Sprintf("No hx_areas found (err: %v)", err),
		)
	}

	for _, hxArea := range hxAreas {
		mustActNow := time.Now().After(hxArea.NextAction)
		slog.Info("MONITOR",
			"area", hxArea.Name,
			"nextAction", hxArea.NextAction,
			"numberName", hxArea.NumberName,
			"mustActNow", mustActNow,
			"lastActionSuccess", hxArea.LastActionSuccess,
		)

		if mustActNow {
			// Check if this number is not already queued for action
			b, _ := areaIsBeingProcessed(hxArea)
			if !b {
				if !hxArea.LastActionSuccess {
					areaFails := incrementAreaFails(hxArea.Name)
					if areaFails >= maxFailsPerArea {
						slog.Warn("MONITOR",
							"message", "Have exceeded the max amount of retries for area",
							"areaName", hxArea.Name,
							"fails", areaFails,
							"maxFails", maxFailsPerArea,
							"skip", true,
						)
					}

					continue
				}

				number, err := db.GetDocument[models.Number]("numbers", bson.M{"name": hxArea.NumberName})
				if err != nil {
					slog.Error("MONITOR",
						"message", fmt.Sprintf("Could not enumerate number '%s'", hxArea.NumberName),
						"error", err.Error(),
					)
					continue
				}

				// Call and set last_action
				slog.Info("MONITOR",
					"action", "call",
					"numberName", hxArea.NumberName,
					"number", number[0].Number,
				)
				initCallAndTranscription(number[0].Number)

				db.UpdateDocument(
					"hx_areas",
					bson.M{"_id": hxArea.ID},
					bson.D{{"$set",
						bson.D{{"last_action", time.Now()}},
					}},
				)

				// Updating the rest of the area is being handled by the callback module
			}
		} else {
			removeAreaFails(hxArea.Name)
		}
	}
}
