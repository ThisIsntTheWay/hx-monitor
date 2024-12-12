package callback

import (
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
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
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
			Fullname: area.Fullname,
			Name:     area.Name,
			Status:   false,
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

// Creates HX sub areas based on an AirspaceStatus for a reference area
func createHxSubAreas(airspaceStatus models.AirspaceStatus, referenceArea string) []models.HXSubArea {
	var result []models.HXSubArea

	for _, area := range airspaceStatus.Areas {
		var subArea models.HXSubArea
		var areaType string
		if area.Index > 0 {
			areaType = "TMA"
		} else {
			areaType = "CTR"
		}

		// Assemble name based on GeoJSON format for object names: <Type> <Area> [Index] HX
		// Examples: TMA Meiringen 1 HX/CTR Meiringen HX

		// Capitalizes first letter (i.e. "meiringen" -> "Meiringen")
		adjustedRefArea := cases.Title(language.English, cases.NoLower).String(referenceArea)
		fullName := fmt.Sprintf("%s %s", areaType, adjustedRefArea)
		if area.Index > 0 {
			fullName = fmt.Sprintf("%s %d", fullName, area.Index)
		}
		fullName = fullName + " HX"

		name := strings.Replace(strings.ToLower(fullName), " ", "-", -1)

		subArea.Fullname = fullName
		subArea.Name = name
		subArea.Status = area.Status

		result = append(result, subArea)
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
func UpdateHxAreaInDatabase(finalTranscript string, callSid string, timestamp time.Time) error {
	// Update HX area
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

	airspaceStatus, err := transcript.ParseTranscript(finalTranscript, timestamp)
	slog.Debug("CALLBACK", "event", "generatedAirspaceStatus", "airspaceStatus", airspaceStatus)

	success, lastError := true, ""
	if err != nil {
		success, lastError = false, err.Error()
	}

	area.NextAction = airspaceStatus.NextUpdate
	area.FlightOperatingHours = airspaceStatus.OperatingHours
	area.SubAreas = createHxSubAreas(airspaceStatus, area.Name)
	area.LastActionSuccess = success
	area.LastError = lastError

	err = db.UpdateDocument(
		"hx_areas",
		bson.D{{"_id", area.ID}},
		bson.D{{"$set", area}},
	)

	return err
}
