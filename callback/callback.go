package callback

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"sort"
	"strconv"
	"time"

	"github.com/thisisnttheway/hx-checker/db"
	"github.com/thisisnttheway/hx-checker/logger"
	"github.com/thisisnttheway/hx-checker/models"
	"github.com/thisisnttheway/hx-checker/transcript"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.ngrok.com/ngrok"
	"golang.ngrok.com/ngrok/config"
)

var CallbackUrl string
var UsePartialTranscriptionResults bool

var _transcriptionRequests []TranscriptionRequest
var statusCallbacks []StatusCallback

// To prevent mapCallSidToNumber() from failing, at the very least 'initiated' can't be ignored
var ignoreCallStates = []string{"queued", "ringing", "in-progress"}
var badCallStates = []string{"busy", "no-answer", "canceled", "failed"}

type StatusCallback struct {
	CallSID        string
	Direction      string
	From           string
	To             string
	CallStatus     string
	SequenceNumber int8
	CallbackSource string
	Duration       int8      // only when status = completed
	Timestamp      time.Time // RFC1123
}

type TranscriptionRequest struct {
	LanguageCode       string            `json:"LanguageCode"`
	TranscriptionSid   string            `json:"TranscriptionSid"`
	PartialResults     bool              `json:"PartialResults"`
	TranscriptionEvent string            `json:"TranscriptionEvent"`
	CallSid            string            `json:"CallSid"`
	TranscriptionData  TranscriptionData `json:"TranscriptionData"`
	Timestamp          time.Time         `json:"Timestamp"`
	AccountSid         string            `json:"AccountSid"`
	Track              string            `json:"Track"`
	Final              bool              `json:"Final"`
	SequenceId         int               `json:"SequenceId"`
	IsInterim          bool              `json:"isInterim"` // Non-standard field
}

type TranscriptionData struct {
	Transcript string  `json:"transcript"`
	Confidence float64 `json:"confidence"`
}

func init() {
	v, exists := os.LookupEnv("TWILIO_PARTIAL_TRANSCRIPTIONS")
	if exists {
		var err error
		UsePartialTranscriptionResults, err = strconv.ParseBool(v)
		if err != nil {
			slog.Error("CALLBACK", "message", "Was unable to parse env var 'TWILIO_PARTIAL_TRANSCRIPTIONS'", "error", err)
		}
	}

	slog.Info("CALLBACK", "event", "init", "usePartialTranscriptionResults", UsePartialTranscriptionResults)
}

func UsesPartialTranscriptionResults() bool {
	return UsePartialTranscriptionResults
}

func GetStatusCallbackurl() string {
	return CallbackUrl
}

func IsCallbackurlSet() bool {
	return CallbackUrl != ""
}

func handleCallsCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Check if request has been sent from Tilio
	twilioSignature := r.Header["X-Twilio-Signature"]
	//userAgent := r.Header["User-Agent"] // TwilioProxy/1.1
	if twilioSignature == nil {
		http.Error(w, "Denied callback", http.StatusForbidden)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	statusCallback := StatusCallback{
		CallSID:        r.FormValue("CallSid"),
		Direction:      r.FormValue("Direction"),
		From:           r.FormValue("From"),
		To:             r.FormValue("To"),
		CallStatus:     r.FormValue("CallStatus"),
		CallbackSource: r.FormValue("CallbackSource"),
	}

	// DB object
	insertObj := models.Call{
		ID:     primitive.NewObjectID(),
		SID:    statusCallback.CallSID,
		Status: statusCallback.CallStatus,
	}

	var timeToUse time.Time = time.Now()
	t, err := time.Parse(time.RFC1123, r.FormValue("Timestamp"))
	if err != nil {
		slog.Error("CALLBACK", "action", "parseFormTimestamp", "error", err)
	} else {
		timeToUse = t
	}

	insertObj.Time = timeToUse

	if r.FormValue("SequenceNumber") != "" {
		sn, err := strconv.ParseInt(r.FormValue("SequenceNumber"), 10, 8)
		if err != nil {
			slog.Error("CALLBACK", "action", "convertSequenceNumber", "source", r.FormValue("SequenceNumber"), "error", err)
		} else {
			statusCallback.SequenceNumber = int8(sn)
		}
	}

	if statusCallback.CallStatus == "completed" {
		insertObj.Time = statusCallback.Timestamp
		convertedDuration, err := strconv.ParseInt(r.FormValue("Duration"), 10, 8)
		if err != nil {
			slog.Error("CALLBACK", "action", "convertCallDuration", "source", r.FormValue("Duration"), "error", err)
			convertedDuration = 0
		}
		statusCallback.Duration = int8(convertedDuration)
	} else if slices.Contains(badCallStates, statusCallback.CallStatus) {
		slog.Error("CALLBACK", "callSid", statusCallback.CallSID, "status", statusCallback.CallStatus, "action", "requeue")

		// Update area accordingly
		const action = "setBadHxStatus"
		n, err := mapCallSidToNumber(statusCallback.CallSID)
		if err != nil {
			slog.Error("CALLBACK", "action", action, "error", err)
		}

		h, err := mapNumberNameToHxArea(n.Name)
		if err != nil {
			slog.Error("CALLBACK", "action", action, "error", err)
		}

		err = setBadHxStatus(h.Name)
		if err != nil {
			slog.Error("CALLBACK", "action", action, "error", err)
		} else {
			slog.Info("CALLBACK", "action", action, "success", true)
		}
	}

	slog.Info("CALLBACK", "event", "receivedEvent", "statusCallback", statusCallback)

	statusCallbacks = append(statusCallbacks, statusCallback)

	var numbers []models.Number
	numbers, dbErr := searchDbForNumber(statusCallback.To)
	if dbErr != nil {
		slog.Error("CALLBACK", "message", "Could not obtain number for given 'TO'", "error", dbErr, "numberTo", statusCallback.To)
	} else {
		insertObj.NumberID = numbers[0].ID
	}

	doDbInsert := !slices.Contains(ignoreCallStates, statusCallback.CallStatus)
	if doDbInsert {
		err := db.InsertDocument("calls", insertObj)
		if err != nil {
			slog.Error("CALLBACK", "message", "Could not insert given statusCallback into DB", "error", err)
		}
	}
}

func handleTransciptionsCallback(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	// Check if request has been sent from Tilio
	twilioSignature := r.Header["X-Twilio-Signature"]
	if twilioSignature == nil {
		http.Error(w, "Denied callback", http.StatusForbidden)
		return
	}

	var transcription TranscriptionRequest
	transcription.LanguageCode = r.FormValue("LanguageCode")
	transcription.TranscriptionSid = r.FormValue("TranscriptionSid")
	transcription.TranscriptionEvent = r.FormValue("TranscriptionEvent")
	transcription.CallSid = r.FormValue("CallSid")

	parsedTime, e := time.Parse(time.RFC3339, r.FormValue("Timestamp"))
	if e != nil {
		parsedTime = time.Now()
	}
	transcription.Timestamp = parsedTime

	transcription.Track = r.FormValue("Track")
	transcription.Final = r.FormValue("Final") == "true"
	transcription.SequenceId = int(r.FormValue("SequenceId")[0])

	if transcription.TranscriptionEvent == "transcription-content" {
		var transcriptData TranscriptionData
		err := json.Unmarshal([]byte(r.FormValue("TranscriptionData")), &transcriptData)
		if err != nil {
			slog.Error("CALLBACK", "message", "Failed json.Unmarshal on interim transcription request", "error", err)
		} else {
			/*
				If we are expecting partial results, then...
				- Assume all transcription JSONs without a "confidence" field are interim results
				  - Ones with such a field are complete transcription segments
				- Only keep the last interim transcript as that will be the most complete sentence

				Very often, Twilio will return one completely transcribed sentence, but then never provide another complete transcription.
				Instead of a complete sentence, a "transcription-stop" event gets sent.
			*/
			if UsesPartialTranscriptionResults() {
				isInterim := transcriptData.Confidence == 0
				transcription.IsInterim = isInterim
			}
		}

		transcription.TranscriptionData = transcriptData
	}

	_transcriptionRequests = append(_transcriptionRequests, transcription)

	var logFields []interface{}
	logFields = append(logFields, "event", transcription.TranscriptionEvent)
	logFields = append(logFields, "transcriptionSid", transcription.TranscriptionSid)
	logFields = append(logFields, "callSid", transcription.CallSid)
	logFields = append(logFields, "sequenceId", transcription.SequenceId)

	// Handle event types
	var isFinalTranscript bool = false
	switch transcription.TranscriptionEvent {
	case "transcription-started":
		logFields = append(logFields, "startTime", transcription.Timestamp)
	case "transcription-content":
		logFields = append(logFields, "transcriptionContent", transcription.TranscriptionData)
	case "transcription-stopped":
		logFields = append(logFields, "endTime", transcription.Timestamp)
		isFinalTranscript = true
	}

	if isFinalTranscript {
		if UsesPartialTranscriptionResults() {
			_transcriptionRequests = sanitizePartialTranscriptions(_transcriptionRequests)
		}

		finalTranscript := handleTranscriptionStopped(transcription)
		logFields = append(logFields, "finalTranscript", finalTranscript)

		airspaceStatus := transcript.ParseTranscript(finalTranscript, time.Now())
		slog.Debug("CALLBACK", "event", "generatedAirspaceStatus", "airspaceStatus", airspaceStatus)

		// Update HX area
		// 1. Get CallSid -> Get Number -> Get HXArea
		// 2. Get HXAreas -> Update them
		// 2. Update hx_areas and hx_sub_areas in DB
		number, err := mapCallSidToNumber(transcription.CallSid)
		if err != nil {
			slog.Error("CALLBACK", "action", "mapCallSidToNumber", "callSid", transcription.CallSid, "error", err)
		}

		area, err := mapNumberNameToHxArea(number.Name)
		if err != nil {
			slog.Error("CALLBACK", "action", "mapNumberNameToHxArea", "numberName", number.Name, "error", err)
		}

		o, _ := json.Marshal(area)
		fmt.Println(string(o))

		// Update DB
		transcriptDbObj := models.Transcript{
			ID:       primitive.NewObjectID(),
			Date:     transcription.Timestamp,
			NumberID: number.ID,
			HXAreaID: area.ID,
			CallSID:  transcription.CallSid,
		}
		err = db.InsertDocument("transcripts", transcriptDbObj)
		if err != nil {
			slog.Error("CALLBACK", "action", "insertTranscriptIntoDatabase", "error", err)
		}

		area.NextAction = airspaceStatus.NextUpdate
		area.SubAreas = createHxSubAreas(airspaceStatus, area.Name)
		area.LastActionSuccess = true

		err = db.UpdateDocument(
			"hx_areas",
			bson.D{{"_id", area.ID}},
			bson.D{{"$set", area}},
		)
		if err != nil {
			slog.Error("CALLBACK", "action", "updateHxAreasInDatabase", "error", err)
		}
	}

	slog.Info("CALLBACK", logFields...)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Event received"))
}

// Assemble a completed transcription by its individual parts and return it
func handleTranscriptionStopped(finalTranscription TranscriptionRequest) string {
	// Filter array for all items whose "callSid" matches the final transcription's callSid and sort
	var transcriptionContents []TranscriptionRequest
	for _, request := range _transcriptionRequests {
		if request.CallSid == finalTranscription.CallSid {
			if request.TranscriptionEvent == "transcription-content" {
				transcriptionContents = append(transcriptionContents, request)
			}
		}
	}

	// Sort by timestamps in ascending order
	sort.Slice(transcriptionContents, func(i, j int) bool {
		return transcriptionContents[i].Timestamp.Before(transcriptionContents[j].Timestamp)
	})

	var fullTranscription string
	for _, t := range transcriptionContents {
		fullTranscription += t.TranscriptionData.Transcript
	}

	// Delete all requests with the same callSid and reassemble array
	var remainingRequests []TranscriptionRequest
	for _, request := range _transcriptionRequests {
		if request.CallSid != finalTranscription.CallSid {
			remainingRequests = append(remainingRequests, request)
		}
	}
	_transcriptionRequests = remainingRequests

	return fullTranscription
}

// Start callback webserver
func Serve() {
	http.HandleFunc("/call", handleCallsCallback)
	http.HandleFunc("/transcription", handleTransciptionsCallback)
	slog.Info("CALLBACK", "message", "Starting webserver")

	_, exists := os.LookupEnv("NGROK_AUTHTOKEN")
	if exists {
		slog.Info("CALLBACK", "message", "Will use ngrok")
		listener, err := ngrok.Listen(
			context.Background(),
			config.HTTPEndpoint(),
			ngrok.WithAuthtokenFromEnv(),
		)
		if err != nil {
			logger.LogErrorFatal("CALLBACK", fmt.Sprintf("Error with ngrok: %v", err.Error()))
		}

		// Set ngrok tunnel URL as CallbackUrl
		CallbackUrl = listener.URL()
		slog.Info("CALLBACK", "callbackUrl", CallbackUrl)
		if err := http.Serve(listener, nil); err != nil {
			logger.LogErrorFatal("CALLBACK", err.Error())
		}
	} else {
		CallbackUrl, exists = os.LookupEnv("TWILIO_API_CALLBACK_URL")
		if !exists {
			logger.LogErrorFatal("CALLBACK", "Must set TWILIO_API_CALLBACK_URL or use ngrok")
		} else {
			slog.Info("CALLBACK", "callbackUrlSource", "envVar", "value", CallbackUrl)
		}

		slog.Info("CALLBACK", "callbackUrl", CallbackUrl)
		if err := http.ListenAndServe(":8080", nil); err != nil {
			logger.LogErrorFatal("CALLBACK", err.Error())
		}
	}
}
