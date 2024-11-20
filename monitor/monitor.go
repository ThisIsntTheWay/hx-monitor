package monitor

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/thisisnttheway/hx-checker/caller"
	"github.com/thisisnttheway/hx-checker/db"
	"github.com/thisisnttheway/hx-checker/logger"
	"github.com/thisisnttheway/hx-checker/models"
	"go.mongodb.org/mongo-driver/bson"
)

type ActionableNumber struct {
	NumberName string
	MustActNow bool
}

// Monitor HX areas
func MonitorHxAreas() {
	hxAreas, err := db.GetDocument[models.HXArea]("hx_areas", bson.D{})
	if len(hxAreas) == 0 || err != nil {
		logger.LogErrorFatal("MONITOR", "No hx_areas found")
	}

	var actionableNumbers []ActionableNumber
	for _, hxArea := range hxAreas {
		mustActNow := time.Now().After(hxArea.NextAction)
		slog.Info(
			"MONITOR",
			"area", hxArea.Name,
			"nextAction", hxArea.NextAction,
			"numberName", hxArea.NumberName,
			"mustActNow", mustActNow,
		)

		if mustActNow {
			// Check if this number is not already queued for action
			isQueued := false
			for _, a := range actionableNumbers {
				if a.NumberName == hxArea.NumberName && a.MustActNow {
					isQueued = true
					slog.Debug("MONITOR", "numberName", a.NumberName, "skipped", true, "reason", "Already queued")
					break
				}
			}

			if isQueued {
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

			actionableNumbers = append(actionableNumbers, ActionableNumber{hxArea.NumberName, mustActNow})

			slog.Info("MONITOR", "numberName", hxArea.NumberName, "number", number[0].Number, "action", "call")
			InitCallAndTranscription(number[0].Number)

			// Set last action
			db.UpdateDocument(
				"hx_areas",
				bson.M{"_id": hxArea.ID},
				bson.D{{"$set",
					bson.D{{"last_action", time.Now()}},
				}},
			)
		}
	}
}

// Call a number and start transcription
func InitCallAndTranscription(number string) caller.CallResponse {
	call, err := caller.Call(number, true)
	if err != nil {
		slog.Error("MONITOR",
			"message", fmt.Sprintf("Failure calling number '%s'", number),
			"error", err.Error(),
		)

		logger.LogErrorFatal("MONITOR", err.Error())
	}

	return call
}
