package callback

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/thisisnttheway/hx-checker/logger"
	"golang.ngrok.com/ngrok"
	"golang.ngrok.com/ngrok/config"
)

var CallbackUrl string = "http://localhost:8080"
var statusCallbacks []StatusCallback

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

type TranscriptionMessage struct {
	LanguageCode       string `json:"LanguageCode"`
	TranscriptionSid   string `json:"TranscriptionSid"`
	TranscriptionEvent string `json:"TranscriptionEvent"`
	CallSid            string `json:"CallSid"`
	TranscriptionData  string `json:"TranscriptionData"`
	Timestamp          string `json:"Timestamp"`
	Final              string `json:"Final"`
	AccountSid         string `json:"AccountSid"`
	Track              string `json:"Track"`
	SequenceId         string `json:"SequenceId"`
}

type TranscriptionData struct {
	Transcript string  `json:"transcript"`
	Confidence float64 `json:"confidence"`
}

func handleCallsCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Check if request has been sent from Tilio
	twilioSignature := r.Header["X-Twilio-Signature"]
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
	// ToDo: Stop live transcript
}

func handleTransciptionsCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Check if request has been sent from Tilio
	twilioSignature := r.Header["X-Twilio-Signature"]
	if twilioSignature == nil {
		http.Error(w, "Denied callback", http.StatusForbidden)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Unable to read request body", http.StatusBadRequest)
		return
	}

	var transcriptionMessage TranscriptionMessage
	if err := json.Unmarshal(body, &transcriptionMessage); err != nil {
		slog.Error("CALLBACK", "message", "Received bad/no JSON", "error", err.Error())
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if transcriptionMessage.TranscriptionEvent == "transcription-content" {
		var transcriptionData TranscriptionData
		if err := json.Unmarshal([]byte(transcriptionMessage.TranscriptionData), &transcriptionData); err != nil {
			msg := "Failed to parse TranscriptionData"
			slog.Error("CALLBACK", "message", msg, "error", err.Error())
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		slog.Info(
			"CALLBACK",
			"message", "Got transcript",
			"callSid", transcriptionMessage.CallSid,
			"final", transcriptionMessage.Final,
			"timestamp", transcriptionMessage.Timestamp,
			"confidence", transcriptionData.Confidence,
			"transcript", transcriptionData.Transcript,
		)

		// ToDo: Store transcript in DB
		// ToDo: Analyze transcript

		w.WriteHeader(http.StatusOK)
		return
	} else {
		http.Error(w, "Invalid TranscriptionEvent", http.StatusBadRequest)
		return
	}
}

// Start callback webserver
func Serve() {
	http.HandleFunc("/call", handleCallsCallback)
	http.HandleFunc("/transcription", handleTransciptionsCallback)
	slog.Info("CALLBACK", "message", "Starting webserver")

	_, exists := os.LookupEnv("NGROK_AUTHTOKEN")
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
		CallbackUrl = listener.URL()
		slog.Info("CALLBACK", "callbackUrl", CallbackUrl)

		if err := http.Serve(listener, nil); err != nil {
			logger.LogErrorFatal("CALLBACK", err.Error())
		}
	} else {
		CallbackUrl = os.Getenv("TWILIO_API_CALLBACK_URL")
		slog.Info("CALLBACK", "callbackUrl", CallbackUrl)

		if err := http.ListenAndServe(":8080", nil); err != nil {
			logger.LogErrorFatal("CALLBACK", err.Error())
		}
	}
}
