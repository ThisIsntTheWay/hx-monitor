package callback

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/thisisnttheway/hx-monitor/db"
	"github.com/thisisnttheway/hx-monitor/models"
	"github.com/thisisnttheway/hx-monitor/transcript"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Searches the DB for a number
func searchDbForNumber(numberTo string) ([]models.Number, error) {
	var result []models.Number
	result, err := db.GetDocument[models.Number]("numbers", bson.M{"number": numberTo})
	if err != nil {
		return result, err
	}

	return result, nil
}

// Maps a call SID to a number
func mapCallSidToNumber(callSid string) (models.Number, error) {
	var numbers []models.Number
	var calls []models.Call
	var err error

	calls, err = db.GetDocument[models.Call]("calls", bson.M{"sid": callSid})
	if err != nil {
		return models.Number{}, err
	}

	numbers, err = db.GetDocument[models.Number](
		"numbers",
		bson.M{"_id": calls[0].NumberID},
	)
	if err != nil {
		return models.Number{}, err
	}

	return numbers[0], nil
}

// Maps a number_name to an hx_area
func mapNumberNameToHxArea(numberName string) (models.HXArea, error) {
	var result []models.HXArea
	result, err := db.GetDocument[models.HXArea](
		"hx_areas",
		bson.M{"number_name": numberName},
	)
	if err != nil {
		return models.HXArea{}, err
	}

	if len(result) > 1 {
		slog.Warn(
			"CALLBACK",
			"message", "Multiple hx_areas for given numberName. Will use first result.",
			"amount", len(result),
			"numberName", numberName,
		)
	}

	return result[0], nil
}

// Sets an HX area to be bad, i.e. all sub areas being false and last action success being false
func setBadHxStatus(referenceArea string, errorReason string) error {
	referenceAreaObj, err := db.GetDocument[models.HXArea]("hx_areas", bson.M{"name": referenceArea})
	if err != nil {
		return err
	}

	var subAreas []models.HXSubArea
	for _, area := range referenceAreaObj[0].SubAreas {
		subAreas = append(subAreas, models.HXSubArea{
			FullName: area.FullName,
			Name:     area.Name,
			Active:   true,
		})
	}

	referenceAreaObj[0].SubAreas = subAreas
	referenceAreaObj[0].LastActionSuccess = false
	referenceAreaObj[0].LastError = errorReason

	err = db.UpdateDocument(
		"hx_areas",
		bson.D{{"_id", referenceAreaObj[0].ID}},
		bson.D{{"$set", referenceAreaObj[0]}},
	)

	return err
}

// Creates HX sub areas for Meiringen
func createHxSubAreasMeiringen(airspaceStatus models.AirspaceMeiringenStatus, referenceArea string) []models.HXSubArea {
	var result []models.HXSubArea

	areasMap := make(map[string]bool)
	areasMap["CTR"] = airspaceStatus.Areas.CTR
	areasMap["TMA1"] = airspaceStatus.Areas.TMA1
	areasMap["TMA2"] = airspaceStatus.Areas.TMA2
	areasMap["TMA3"] = airspaceStatus.Areas.TMA3
	areasMap["TMA4"] = airspaceStatus.Areas.TMA4
	areasMap["TMA5"] = airspaceStatus.Areas.TMA5
	areasMap["TMA6"] = airspaceStatus.Areas.TMA6

	// The name for HxSubArea must be "<type> <areaName> [index] HX" to be able to transformed correctly.
	// The frontend will expect these keys to be formatted in a particular way.
	for k, active := range areasMap {
		var fullName string
		if k == "CTR" {
			fullName = "CTR Meiringen HX"
		} else {
			fullName = fmt.Sprintf("TMA Meiringen %s HX", strings.TrimPrefix(k, "TMA"))
		}
		result = append(result, models.HXSubArea{
			FullName: fullName,
			Name:     strings.ReplaceAll(strings.ToLower(fullName), " ", "-"),
			Active:   active,
		})
	}

	return result
}

// Removes all interim transcripts with the exception of the very last one
func sanitizePartialTranscriptions(s []TranscriptionCallback) []TranscriptionCallback {
	// Partial transcripts all have the same sequence ID, but different timestamps
	// As such, we'll have to resort to sorting by timestamps
	sort.Slice(s, func(i, j int) bool {
		return s[i].Timestamp.Before(s[j].Timestamp)
	})

	// First, remove all interim entries that do not meet the minimal length
	const minLength int = 16
	var intermediateResults []TranscriptionCallback
	for _, entry := range s {
		entry.IsInterim = entry.TranscriptionData.Confidence == 0
		if len(entry.TranscriptionData.Transcript) >= minLength || !entry.IsInterim {
			intermediateResults = append(intermediateResults, entry)
		}
	}

	// Secondly, remove all interim entries but keep track of the last entry
	var result []TranscriptionCallback
	var lastInterim *TranscriptionCallback
	for _, entry := range intermediateResults {
		if entry.TranscriptionData.Confidence == 0 {
			lastInterim = &entry
		} else {
			result = append(result, entry)
		}
	}

	// Append last inerim entry to final result
	if lastInterim != nil {
		result = append(result, *lastInterim)
	}

	return result
}

// Updates an HX area in DB based on parsed transcript data
// Important: Only equipped to handle meiringen at this moment
func UpdateHxAreaInDatabase(finalTranscript string, callSid string, timestamp time.Time) error {
	ctx := context.TODO()

	// 1. Get CallSid -> Get Number -> Get HXArea
	// 2. Get HXAreas -> Update them
	// 2. Update hx_areas and hx_sub_areas in DB
	number, err := mapCallSidToNumber(callSid)
	if err != nil {
		slog.Error("CALLBACK", "action", "mapCallSidToNumber", "callSid", callSid, "error", err)
	}

	area, err := mapNumberNameToHxArea(number.Name)
	if err != nil {
		slog.Error("CALLBACK", "action", "mapNumberNameToHxArea", "numberName", number.Name, "error", err)
	}

	// Update DB
	transcriptDbObj := models.Transcript{
		ID:         primitive.NewObjectID(),
		Transcript: finalTranscript,
		Date:       timestamp,
		NumberID:   number.ID,
		HXAreaID:   area.ID,
		CallSID:    callSid,
	}
	err = db.InsertDocument("transcripts", transcriptDbObj)
	if err != nil {
		slog.Error("CALLBACK", "action", "insertTranscriptIntoDatabase", "error", err)
	}

	success, lastError := true, ""

	// ToDo, once other parsers are set up: Determine what parser to use based on phone number
	airspaceStatus, err := transcript.ParseAirspaceTranscriptMeiringen(finalTranscript, ctx)
	slog.Debug("CALLBACK", "event", "generatedAirspaceStatus", "airspaceStatus", airspaceStatus)
	if err != nil {
		success, lastError = false, err.Error()
	}
	area.SubAreas = createHxSubAreasMeiringen(airspaceStatus, area.Name)

	area.LastActionSuccess = success
	area.LastError = lastError

	err = db.UpdateDocument(
		"hx_areas",
		bson.D{{"_id", area.ID}},
		bson.D{{"$set", area}},
	)

	return err
}
