package monitor

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/thisisnttheway/hx-monitor/caller"
	"github.com/thisisnttheway/hx-monitor/db"
	"github.com/thisisnttheway/hx-monitor/logger"
	"github.com/thisisnttheway/hx-monitor/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type ActionableNumber struct {
	NumberName string
	MustActNow bool
}

type CallConfiguration struct {
	DoTranscription bool
	DoRecording     bool
}

var _callConfiguration CallConfiguration

// { "<area>": <num_fails> }
var _areaFailureCounts map[string]int8 = make(map[string]int8)

// { "<area>": <being_processed> }
var _areaProcessingQueue map[string]bool = make(map[string]bool)

var maxFailsPerArea int8 = 3
var onErrorNextActionDelay time.Duration = 30 * time.Minute

func init() {
	// Looks up and returns an env vars value as bool. Returns s otherwise.
	var check = func(e string, s bool) bool {
		env, exists := os.LookupEnv(e)
		if exists {
			b, err := strconv.ParseBool(env)
			if err != nil {
				return false
			} else {
				return b
			}
		} else {
			return s
		}
	}

	t := check("USE_TWILIO_TRANSCRIPTION", true)
	w := check("USE_WHISPER_TRANSCRIPTION", false)
	if t && w {
		logger.LogErrorFatal(
			"MONITOR",
			"Both USE_TWILIO_TRANSCRIPTION and USE_WHISPER_TRANSCRIPTION are set",
		)
	}

	_callConfiguration.DoTranscription = t
	_callConfiguration.DoRecording = w
}

func GetAreaProcessingState(areaName string) bool {
	return _areaProcessingQueue[areaName]
}

func DeleteAreaFromProcessingQueue(areaName string) {
	delete(_areaProcessingQueue, areaName)
}

func setAreaProcessingState(areaName string, state bool) {
	_areaProcessingQueue[areaName] = state
}

// Determines if an area is being processed based on its last_action timestamp and associated, non-completed calls
func areasNumberIsBeingCalled(area models.HXArea) (bool, error) {
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
			"areaName", area.Name,
			"areaLastAction", area.LastAction,
			"areaId", area.ID,
		)
		return false, err
	}

	if len(results) == 0 {
		slog.Warn("MONITOR",
			"action", "aggregateHxAreas",
			"message", "Length of aggregated result is 0",
			"areaName", area.Name,
			"areaLastAction", area.LastAction,
			"areaId", area.ID,
		)

		return false, nil
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
		// Either no calls have ever been made, or calls older than LastAction actually did complete
		hasCompletedCalls = true
		slog.Warn("MONITOR",
			"action", "aggregateHxAreas",
			"message", "Area has had no calls newer than referenceTime",
			"areaName", area.Name,
			"referenceTime", area.LastAction,
			"areaId", area.ID,
		)
	}

	o, _ := json.Marshal(results)
	slog.Debug("MONITOR",
		"action", "aggregateHxAreas",
		"areaName", area.Name,
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
		if v < maxFailsPerArea {
			_areaFailureCounts[areaName] = v + 1
		}
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

// Call a number and either start transcription or recording
func initCall(number string) caller.CallResponse {
	call, err := caller.Call(
		number,
		_callConfiguration.DoTranscription,
		_callConfiguration.DoRecording,
	)
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
			if GetAreaProcessingState(hxArea.Name) {
				slog.Info("MONITOR_DEBUG", "event", "skipAreaDueToProcessingState", "area", hxArea.Name)
				continue
			}

			// Check if this number is not already being called
			b, _ := areasNumberIsBeingCalled(hxArea)
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
						continue
					} else {
						// Delay processing for X amount of time on next run
						newNextAction := time.Now().Add(onErrorNextActionDelay)
						err := db.UpdateDocument(
							"hx_areas",
							bson.M{"_id": hxArea.ID},
							bson.D{{"$set",
								bson.D{{"next_action", newNextAction}},
							}},
						)
						if err != nil {
							slog.Error("MONITOR", "action", "delayNextAction", "error", err)
						}
					}
				}

				setAreaProcessingState(hxArea.Name, true)

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
				initCall(number[0].Number)

				db.UpdateDocument(
					"hx_areas",
					bson.M{"_id": hxArea.ID},
					bson.D{{"$set",
						bson.D{{"last_action", time.Now()}},
					}},
				)

				// Updating the rest of the area is being handled by the callback module
			} else {
				slog.Info("MONTOR",
					"action", "scheduleCall",
					"skip", true,
					"areaName", hxArea.Name,
				)
			}
		} else {
			removeAreaFails(hxArea.Name)
			setAreaProcessingState(hxArea.Name, false)
		}
	}
}
