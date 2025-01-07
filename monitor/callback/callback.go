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
	"strings"
	"time"

	"github.com/thisisnttheway/hx-monitor/caller"
	c "github.com/thisisnttheway/hx-monitor/configuration"
	"github.com/thisisnttheway/hx-monitor/db"
	"github.com/thisisnttheway/hx-monitor/logger"
	"github.com/thisisnttheway/hx-monitor/models"
	"github.com/thisisnttheway/hx-monitor/transcript"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.ngrok.com/ngrok"
	"golang.ngrok.com/ngrok/config"
)

var _TranscriptionCallbacks []TranscriptionCallback
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

type TranscriptionCallback struct {
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

type RecordingCallback struct {
	CallSid         string `json:"CallSid"`
	RecordingSid    string `json:"RecordingSid"`
	RecordingStatus string `json:"RecordingStatus"`
	RecordingUrl    string `json:"RecordingUrl"`
}

func init() {
	v, exists := os.LookupEnv("TWILIO_PARTIAL_TRANSCRIPTIONS")
	if exists {
		var result bool
		v, err := strconv.ParseBool(v)
		if err != nil {
			slog.Error("CALLBACK", "message", "Was unable to parse env var 'TWILIO_PARTIAL_TRANSCRIPTIONS'", "error", err)
		} else {
			result = v
		}

		c.SetPartialTranscriptionResultBool(result)
	}

	slog.Info("CALLBACK", "event", "init", "TWILIO_PARTIAL_TRANSCRIPTIONS", c.UsesPartialTranscriptionResults())
}

// Handler for /call
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

	// Fallback timestamp
	var timeToUse time.Time = time.Now()
	t, err := time.Parse(time.RFC1123, r.FormValue("Timestamp"))
	if err != nil {
		slog.Error("CALLBACK", "action", "parseFormTimestamp", "error", err)
	} else if !t.Equal(time.Unix(0, 0)) {
		slog.Info("CALLBACK", "action", "parseFormTimestamp", "parsedTimestamp", t)
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

		err = setBadHxStatus(h.Name, err.Error())
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

// Handler for /transcription
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

	var transcription TranscriptionCallback
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
			if c.UsesPartialTranscriptionResults() {
				isInterim := transcriptData.Confidence == 0
				transcription.IsInterim = isInterim
			}
		}

		transcription.TranscriptionData = transcriptData
	}

	_TranscriptionCallbacks = append(_TranscriptionCallbacks, transcription)

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
		if c.UsesPartialTranscriptionResults() {
			_TranscriptionCallbacks = sanitizePartialTranscriptions(_TranscriptionCallbacks)
		}

		finalTranscript := handleTranscriptionStopped(transcription)
		logFields = append(logFields, "finalTranscript", finalTranscript)

		err := UpdateHxAreaInDatabase(
			finalTranscript,
			transcription.CallSid,
			transcription.Timestamp,
		)
		if err != nil {
			slog.Error("CALLBACK", "event", "updateHxAreaInDatabase", "error", err)
		}
	}

	slog.Info("CALLBACK", logFields...)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Event received"))
}

// Handler for /recording
func handleRecordingsCallback(w http.ResponseWriter, r *http.Request) {
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

	var recording RecordingCallback
	recording.CallSid = r.FormValue("CallSid")
	recording.RecordingSid = r.FormValue("RecordingSid")
	recording.RecordingStatus = r.FormValue("RecordingStatus")
	recording.RecordingUrl = r.FormValue("RecordingUrl")

	if strings.ToLower(recording.RecordingStatus) == "completed" {
		filePath, err := DownloadRecording(
			recording.RecordingSid,
			recording.RecordingUrl,
		)
		if err != nil {
			slog.Error("CALLBACK",
				"action", "downloadRecording",
				"error", err,
			)
		} else {
			slog.Info("CALLBACK",
				"event", "downloadRecordingComplete",
				"filePath", filePath,
			)
		}

		err = caller.DeleteRecording(recording.RecordingSid)
		if err != nil {
			slog.Warn("CALLER",
				"action", "deleteRecording",
				"error", err,
			)
		}

		// Transcribe recording with whisper
		finalTranscript, err := transcript.Transcribe(filePath)
		if err != nil {
			logger.LogErrorFatal("CALLBACK", fmt.Sprintf("Failed to transcribe recording: %v", err))
		}

		err = UpdateHxAreaInDatabase(
			finalTranscript,
			recording.CallSid,
			time.Now(),
		)
		if err != nil {
			logger.LogErrorFatal("CALLBACK", fmt.Sprintf("Failed UpdateHxAreaInDatabase: %v", err))
		}
	}
}

// Assemble a completed transcription by its individual parts and return it
func handleTranscriptionStopped(finalTranscription TranscriptionCallback) string {
	// Filter array for all items whose "callSid" matches the final transcription's callSid and sort
	var transcriptionContents []TranscriptionCallback
	for _, request := range _TranscriptionCallbacks {
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
	var remainingRequests []TranscriptionCallback
	for _, request := range _TranscriptionCallbacks {
		if request.CallSid != finalTranscription.CallSid {
			remainingRequests = append(remainingRequests, request)
		}
	}
	_TranscriptionCallbacks = remainingRequests

	return fullTranscription
}

// Start callback webserver
func Serve() {
	http.HandleFunc(c.UrlConfigs.Calls, handleCallsCallback)
	http.HandleFunc(c.UrlConfigs.Transcriptions, handleTransciptionsCallback)
	http.HandleFunc(c.UrlConfigs.Recordings, handleRecordingsCallback)

	// ngrok automatically uses the env var so no need to pass the actual value anywhere
	_, exists := os.LookupEnv("NGROK_AUTHTOKEN")
	slog.Info("CALLBACK", "action", "startWebserver", "useNgrok", exists)
	if exists {
		listener, err := ngrok.Listen(
			context.Background(),
			config.HTTPEndpoint(),
			ngrok.WithAuthtokenFromEnv(),
		)
		if err != nil {
			logger.LogErrorFatal("CALLBACK", fmt.Sprintf("Error with ngrok: %v", err.Error()))
		}

		// Set ngrok tunnel URL as CallbackUrl
		c.SetCallbackUrl(listener.URL())
		slog.Info("CALLBACK", "callbackUrl", c.CallbackUrl)
		if err := http.Serve(listener, nil); err != nil {
			logger.LogErrorFatal("CALLBACK", err.Error())
		}
	} else {
		customCallbackUrl, exists := os.LookupEnv("TWILIO_API_CALLBACK_URL")
		if !exists {
			logger.LogErrorFatal("CALLBACK", "Must set TWILIO_API_CALLBACK_URL or use ngrok")
		} else {
			c.SetCallbackUrl(customCallbackUrl)
			slog.Info("CALLBACK", "callbackUrlSource", "envVar", "value", c.CallbackUrl)
		}

		slog.Info("CALLBACK", "callbackUrl", c.CallbackUrl)
		if err := http.ListenAndServe(":8080", nil); err != nil {
			logger.LogErrorFatal("CALLBACK", err.Error())
		}
	}
}
