package callback

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/thisisnttheway/hx-checker/logger"
	"github.com/thisisnttheway/hx-checker/transcriptParser"
	"golang.ngrok.com/ngrok"
	"golang.ngrok.com/ngrok/config"
)

var CallbackUrl string
var statusCallbacks []StatusCallback

var _transcriptionRequests []TranscriptionRequest

type StatusCallback struct {
	CallSID        string
	Direction      string
	From           string
	To             string
	CallStatus     string
	SequenceNumber int8
	CallbackSource string
	Duration       int8      // only when status = completed
	Timestamp      time.Time // only when status = completed
}

type TranscriptionRequest struct {
	LanguageCode       string `json:"LanguageCode"`
	TranscriptionSid   string `json:"TranscriptionSid"`
	PartialResults     bool   `json:"PartialResults"`
	TranscriptionEvent string `json:"TranscriptionEvent"`
	CallSid            string `json:"CallSid"`
	TranscriptionData  string `json:"TranscriptionData"`
	Timestamp          string `json:"Timestamp"`
	AccountSid         string `json:"AccountSid"`
	Track              string `json:"Track"`
	Final              bool   `json:"Final"`
	SequenceId         int    `json:"SequenceId"`
}

type TranscriptionData struct {
	Transcript string  `json:"transcript"`
	Confidence float64 `json:"confidence"`
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

	if r.FormValue("SequenceNumber") != "" {
		sn, err := strconv.ParseInt(r.FormValue("SequenceNumber"), 10, 8)
		if err != nil {
			slog.Error("CALLBACK", "message", "Failed converting sequenceNumber", "source", r.FormValue("SequenceNumber"), "error", err.Error())
		} else {
			statusCallback.SequenceNumber = int8(sn)
		}
	}

	if statusCallback.CallStatus == "completed" {
		convertedDuration, err := strconv.ParseInt(r.FormValue("Duration"), 10, 8)
		if err != nil {
			slog.Error("CALLBACK", "message", "Failed converting duration", "source", r.FormValue("Duration"), "error", err.Error())
			convertedDuration = 0
		}
		statusCallback.Duration = int8(convertedDuration)
	}

	slog.Info("CALLBACK", "event", "receivedEvent", "statusCallback", statusCallback)

	statusCallbacks = append(statusCallbacks, statusCallback)

	// ToDo: Insert call into db if completed
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
	transcription.TranscriptionData = r.FormValue("TranscriptionData")
	transcription.Timestamp = r.FormValue("Timestamp")
	transcription.Track = r.FormValue("Track")
	transcription.Final = r.FormValue("Final") == "true"
	transcription.SequenceId = int(r.FormValue("SequenceId")[0])

	_transcriptionRequests = append(_transcriptionRequests, transcription)

	var logFields []interface{}
	logFields = append(logFields, "event", transcription.TranscriptionEvent)
	logFields = append(logFields, "transcriptionSid", transcription.TranscriptionSid)
	logFields = append(logFields, "callSid", transcription.CallSid)

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
		finalTranscript := handleTranscriptionStopped(transcription)
		logFields = append(logFields, "finalTranscript", finalTranscript)

		airspaceStatus := transcriptParser.ParseTranscript(finalTranscript, time.Now())
		o, _ := json.MarshalIndent(airspaceStatus, "", "  ")
		fmt.Println(string(o))
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

	sort.SliceStable(transcriptionContents, func(i, j int) bool {
		return transcriptionContents[i].SequenceId < transcriptionContents[j].SequenceId
	})

	// Assemble transcript
	var fullTranscription string
	for _, content := range transcriptionContents {
		var tr TranscriptionData
		e := json.Unmarshal([]byte(content.TranscriptionData), &tr)
		if e != nil {
			logger.LogErrorFatal("PARSER", fmt.Sprintf("Failure unmarshaling transcription data: %v", e))
		}

		fullTranscription += tr.Transcript
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
			slog.Info("CALLBACK", "message", "Have set callback URL to env var", "value", CallbackUrl)
		}

		slog.Info("CALLBACK", "callbackUrl", CallbackUrl)
		if err := http.ListenAndServe(":8080", nil); err != nil {
			logger.LogErrorFatal("CALLBACK", err.Error())
		}
	}
}
