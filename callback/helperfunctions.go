package callback

import (
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/thisisnttheway/hx-checker/db"
	"github.com/thisisnttheway/hx-checker/models"
	"github.com/thisisnttheway/hx-checker/transcriptParser"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Searches the DB for a number
func searchDbForNumber(numberTo string) ([]models.Number, error) {
	var result []models.Number
	result, err := db.GetDocument[models.Number]("numbers", bson.D{{"number", numberTo}})
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

	calls, err = db.GetDocument[models.Call]("calls", bson.D{{"sid", callSid}})
	if err != nil {
		return models.Number{}, err
	}

	numbers, err = db.GetDocument[models.Number](
		"numbers",
		bson.D{{"_id", calls[0].NumberID}},
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
		bson.D{{"number_name", numberName}},
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

// Creates HX sub areas based on an AirspaceStatus for a reference area
func createHxSubAreas(airspaceStatus transcriptParser.AirspaceStatus, referenceArea string) []models.HXSubArea {
	var result []models.HXSubArea

	for _, area := range airspaceStatus.Areas {
		var subArea models.HXSubArea
		var areaType string
		if area.Index > 0 {
			areaType = "TMA"
		} else {
			areaType = "CTR"
		}

		// Capitalizes first letter (i.e. "meiringen" -> "Meiringen")
		adjustedRefArea := cases.Title(language.English, cases.NoLower).String(referenceArea)
		fullName := fmt.Sprintf("%s %s", adjustedRefArea, areaType)
		if area.Index > 0 {
			fullName = fmt.Sprintf("%s %d", fullName, area.Index)
		}

		name := strings.Replace(strings.ToLower(fullName), " ", "-", -1)

		subArea.Fullname = fullName
		subArea.Name = name
		subArea.Status = area.Status

		result = append(result, subArea)
	}

	return result
}

// Removes all interim transcripts with the exception of the very last one
func sanitizePartialTranscriptions(s []TranscriptionRequest) []TranscriptionRequest {
	// Partial transcripts all have the same sequence ID, but different timestamps
	// As such, we'll have to resort to sorting by timestamps
	sort.Slice(s, func(i, j int) bool {
		return s[i].Timestamp.Before(s[j].Timestamp)
	})

	// First, remove all interim entries that do not meet the minimal length
	const minLength int = 16
	var intermediateResults []TranscriptionRequest
	for _, entry := range s {
		entry.IsInterim = entry.TranscriptionData.Confidence == 0
		if len(entry.TranscriptionData.Transcript) >= minLength || !entry.IsInterim {
			intermediateResults = append(intermediateResults, entry)
		}
	}

	// Secondly, remove all interim entries but keep track of the last entry
	var result []TranscriptionRequest
	var lastInterim *TranscriptionRequest
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
