package monitor

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/thisisnttheway/hx-monitor/caller"
	"github.com/thisisnttheway/hx-monitor/db"
	"github.com/thisisnttheway/hx-monitor/models"
	"github.com/thisisnttheway/hx-monitor/rmq"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type CallConfiguration struct {
	DoTranscription bool
	DoRecording     bool
	MaxRetries      int
}

var (
	_callConfiguration CallConfiguration
)

func init() {
	_callConfiguration.DoTranscription = true
	_callConfiguration.DoRecording = false
	_callConfiguration.MaxRetries = 3
}

// CheckAreaHasActiveCalls checks if an area has messages in the RabbitMQ queue
// This replaces the database-dependent areasNumberIsBeingCalled function
func CheckAreaHasActiveCalls(areaID primitive.ObjectID) (bool, error) {
	// Query MongoDB to see if there are any non-completed calls for this area
	type CallResult struct {
		Count int `bson:"count"`
	}

	results, err := db.Aggregate[CallResult]("calls", mongo.Pipeline{
		bson.D{{"$lookup", bson.D{
			{"from", "numbers"},
			{"localField", "number_id"},
			{"foreignField", "_id"},
			{"as", "number_info"},
		}}},
		bson.D{{"$unwind", "$number_info"}},
		bson.D{{"$lookup", bson.D{
			{"from", "hx_areas"},
			{"localField", "number_info.name"},
			{"foreignField", "number_name"},
			{"as", "area_info"},
		}}},
		bson.D{{"$unwind", "$area_info"}},
		bson.D{{"$match", bson.M{
			"area_info._id": areaID,
			"status":        bson.M{"$ne": "completed"},
		}}},
		bson.D{{"$count", "count"}},
	})

	if err != nil {
		slog.Error("MONITOR", "action", "CheckAreaHasActiveCalls", "areaID", areaID, "error", err)
		return false, err
	}

	if len(results) == 0 {
		return false, nil
	}

	return results[0].Count > 0, nil
}

// initCall calls a number and handles potential failures
// Returns (CallResponse, error) where error indicates if it failed or phone system didn't answer
func initCall(number string) (caller.CallResponse, error) {
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
		return call, err
	}

	// Check if the call was actually answered
	// Status might be "queued", "ringing", "in-progress", "completed", "busy", "failed", "no-answer"
	if call.Status == "no-answer" || call.Status == "busy" || call.Status == "failed" {
		slog.Warn("MONITOR",
			"message", fmt.Sprintf("Phone system did not answer for number '%s'", number),
			"callStatus", call.Status,
			"callSID", call.SID,
		)
		return call, fmt.Errorf("phone system not available: %s", call.Status)
	}

	return call, nil
}

// Monitor HX areas: Keep track of states and publish call tasks via RabbitMQ
func MonitorHxAreas() error {
	hxAreas, err := db.GetDocument[models.HXArea]("hx_areas", bson.D{})
	if len(hxAreas) == 0 || err != nil {
		return fmt.Errorf("no hx_areas found (err: %v)", err)
	}

	for _, hxArea := range hxAreas {
		mustActNow := time.Now().UTC().After(hxArea.NextAction)
		slog.Info("MONITOR",
			"area", hxArea.Name,
			"nextAction", hxArea.NextAction,
			"numberName", hxArea.NumberName,
			"numErrors", hxArea.NumErrors,
			"mustActNow", mustActNow,
			"lastActionSuccess", hxArea.LastActionSuccess,
		)

		if mustActNow {
			// Check if this area already has active calls
			hasActiveCalls, err := CheckAreaHasActiveCalls(hxArea.ID)
			if err != nil {
				slog.Error("MONITOR", "action", "CheckAreaHasActiveCalls", "area", hxArea.Name, "error", err)
			}

			if hasActiveCalls {
				slog.Info("MONITOR", "action", "skipArea", "reason", "activeCalls", "area", hxArea.Name)
				continue
			}

			// If previous action failed and we haven't exceeded max retries, skip for now
			if !hxArea.LastActionSuccess && hxArea.NumErrors >= int8(_callConfiguration.MaxRetries) {
				slog.Warn("MONITOR",
					"message", "Have exceeded the max amount of retries for area",
					"areaName", hxArea.Name,
					"fails", hxArea.NumErrors,
					"maxFails", _callConfiguration.MaxRetries,
					"skip", true,
				)
				continue
			}

			// Get the phone number to call
			number, err := db.GetDocument[models.Number]("numbers", bson.M{"name": hxArea.NumberName})
			if err != nil {
				slog.Error("MONITOR",
					"message", fmt.Sprintf("Could not enumerate number '%s'", hxArea.NumberName),
					"error", err.Error(),
				)
				continue
			}

			if len(number) == 0 {
				slog.Error("MONITOR",
					"message", fmt.Sprintf("Number '%s' not found", hxArea.NumberName),
				)
				continue
			}

			// Publish call task to RabbitMQ
			callTask := rmq.CallTaskMessage{
				AreaID:          hxArea.ID,
				AreaName:        hxArea.Name,
				NumberName:      hxArea.NumberName,
				PhoneNumber:     number[0].Number,
				RetryCount:      int(hxArea.NumErrors),
				MaxRetries:      _callConfiguration.MaxRetries,
				DoTranscription: _callConfiguration.DoTranscription,
				DoRecording:     _callConfiguration.DoRecording,
				Timestamp:       time.Now(),
			}

			if err := rmq.PublishCallTask(callTask); err != nil {
				slog.Error("MONITOR",
					"action", "PublishCallTask",
					"areaName", hxArea.Name,
					"error", err,
				)
				continue
			}

			slog.Info("MONITOR",
				"action", "publishCallTask",
				"numberName", hxArea.NumberName,
				"number", number[0].Number,
				"areaName", hxArea.Name,
			)

			// Update last_action timestamp
			db.UpdateDocument(
				"hx_areas",
				bson.M{"_id": hxArea.ID},
				bson.D{{"$set",
					bson.D{{"last_action", time.Now()}},
				}},
			)
		}
	}

	return nil
}

// ProcessCallTask processes a call task from RabbitMQ
// This worker function handles the actual call and manages retries
func ProcessCallTask(taskMsg rmq.CallTaskMessage) error {
	slog.Info("MONITOR", "action", "ProcessCallTask", "areaName", taskMsg.AreaName, "retry", taskMsg.RetryCount)

	// Make the call
	callResp, err := initCall(taskMsg.PhoneNumber)

	if err != nil {
		slog.Warn("MONITOR",
			"action", "ProcessCallTask",
			"areaName", taskMsg.AreaName,
			"error", err,
			"retryCount", taskMsg.RetryCount,
		)

		// If we haven't exceeded max retries, republish to delayed queue
		if taskMsg.RetryCount < taskMsg.MaxRetries {
			taskMsg.RetryCount++
			if err := rmq.PublishCallTaskDelayed(taskMsg); err != nil {
				slog.Error("MONITOR",
					"action", "PublishCallTaskDelayed",
					"areaName", taskMsg.AreaName,
					"error", err,
				)
			}
			// Update area with new error count
			db.UpdateDocument(
				"hx_areas",
				bson.M{"_id": taskMsg.AreaID},
				bson.D{{"$set",
					bson.D{
						{"num_errors", taskMsg.RetryCount},
						{"last_action_success", false},
						{"last_error", err.Error()},
					},
				}},
			)
		} else {
			// Max retries exceeded, mark as failed
			slog.Error("MONITOR",
				"action", "ProcessCallTask",
				"message", "Max retries exceeded",
				"areaName", taskMsg.AreaName,
				"maxRetries", taskMsg.MaxRetries,
			)
			db.UpdateDocument(
				"hx_areas",
				bson.M{"_id": taskMsg.AreaID},
				bson.D{{"$set",
					bson.D{
						{"last_action_success", false},
						{"last_error", fmt.Sprintf("Max retries exceeded: %v", err)},
					},
				}},
			)
		}

		return err
	}

	slog.Info("MONITOR",
		"action", "ProcessCallTask",
		"areaName", taskMsg.AreaName,
		"callSID", callResp.SID,
		"status", callResp.Status,
	)

	// Publish call completion message
	completed := rmq.CallCompletedMessage{
		AreaID:     taskMsg.AreaID,
		AreaName:   taskMsg.AreaName,
		CallSID:    callResp.SID,
		Status:     "success",
		RetryCount: taskMsg.RetryCount,
		MaxRetries: taskMsg.MaxRetries,
		Timestamp:  time.Now(),
	}

	if err := rmq.PublishCallCompleted(completed); err != nil {
		slog.Error("MONITOR",
			"action", "PublishCallCompleted",
			"areaName", taskMsg.AreaName,
			"error", err,
		)
	}

	// Reset error count on success
	db.UpdateDocument(
		"hx_areas",
		bson.M{"_id": taskMsg.AreaID},
		bson.D{{"$set",
			bson.D{
				{"num_errors", 0},
				{"last_action_success", true},
			},
		}},
	)

	return nil
}
