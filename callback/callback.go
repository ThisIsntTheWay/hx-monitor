package callback

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"

	"github.com/thisisnttheway/hx-checker/logger"
	"golang.ngrok.com/ngrok"
	"golang.ngrok.com/ngrok/config"
)

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

func handleCallback(w http.ResponseWriter, r *http.Request) {
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

		// ToDo: Analyze transcript

		w.WriteHeader(http.StatusOK)
	} else {
		http.Error(w, "Invalid TranscriptionEvent", http.StatusBadRequest)
	}
}

// Start callback webserver and return an ngrok URL, if applicable
func Serve() {
	http.HandleFunc("/callback", handleCallback)
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
		slog.Info("CALLBACK", "ngrokTunnelUrl", listener.URL())
		os.Setenv("TWILIO_API_CALLBACK_URL", listener.URL())

		if err := http.Serve(listener, nil); err != nil {
			logger.LogErrorFatal("CALLBACK", err.Error())
		}
	} else {
		if err := http.ListenAndServe(":8080", nil); err != nil {
			logger.LogErrorFatal("CALLBACK", err.Error())
		}
	}
}
