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
	MustAct    bool
}

// Monitor HX areas
func MonitorHxAreas() {
	hxAreas, err := db.GetDocument[models.HXArea]("hx_areas", bson.D{})
	if len(hxAreas) == 0 || err != nil {
		logger.LogErrorFatal("MONITOR", "No hx_areas found")
	}

	var actionableNumbers []ActionableNumber
	for _, v := range hxAreas {
		mustAct := v.NextAction.Unix() < time.Now().Unix()
		slog.Info("MONITOR", "area", v.Area, "nextAction", v.NextAction, "numberName", v.NumberName, "mustAct", mustAct)

		if mustAct {
			// Check if this number is not already queued for action
			isQueued := false
			for _, a := range actionableNumbers {
				if a.NumberName == v.NumberName && a.MustAct {
					isQueued = true
					slog.Debug("MONITOR", "numberName", a.NumberName, "skipped", true, "reason", "Already queued")
					break
				}
			}

			if isQueued {
				continue
			}

			number, err := db.GetDocument[models.Number]("numbers", bson.D{{"name", v.NumberName}})
			if err != nil {
				slog.Error("MONITOR",
					"message", fmt.Sprintf("Could not enumerate number '%s'", v.NumberName),
					"error", err.Error(),
				)
			}

			actionableNumbers = append(actionableNumbers, ActionableNumber{v.NumberName, mustAct})
			slog.Info("MONITOR", "numberName", v.NumberName, "number", number[0].Number)

			// ToDo: Act
			// CallAndTranscribeNumber(...)
		}
	}
}

// Call a number and start transcription
func InitCallAndTranscription(number string, numberName string) error {
	call, err := caller.Call(number)
	if err != nil {
		slog.Error("MONITOR",
			"message", fmt.Sprintf("Failure calling number '%s' (%s)", number, numberName),
			"error", err.Error(),
		)
	}

	// ToDo: Start transcript

	return nil
}
