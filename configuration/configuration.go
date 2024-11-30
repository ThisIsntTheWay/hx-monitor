package configuration

import (
	"log/slog"
	"os"
	"strconv"

	"github.com/thisisnttheway/hx-checker/logger"
)

// --------------------------
// CALLBACK
type UrlConfig struct {
	Calls          string
	Transcriptions string
	Recordings     string
}

var UrlConfigs UrlConfig = UrlConfig{
	Calls:          "/calls",
	Transcriptions: "/transcription",
	Recordings:     "/recording",
}

var CallbackUrl string

func SetCallbackUrl(value string) {
	CallbackUrl = value
}

func GetCallbackUrl() string {
	return CallbackUrl
}

func IsCallbackurlSet() bool {
	return CallbackUrl != ""
}

// --------------------------
// TWILIO
type TwilioConfiguration struct {
	UsePartialTranscriptionResults bool
	CallLength                     int
	CallFrom                       string
	AuthConfig                     TwilioAuth
}

type TwilioAuth struct {
	AccountSid string
	AuthToken  string
	ApiKey     string
	ApiSecret  string
}

var TwilioConfig TwilioConfiguration

func UsesPartialTranscriptionResults() bool {
	return TwilioConfig.UsePartialTranscriptionResults
}

func SetPartialTranscriptionResultBool(value bool) {
	TwilioConfig.UsePartialTranscriptionResults = value
}

func GetTwilioConfig() TwilioConfiguration {
	return TwilioConfig
}

// --------------------------
// WHISPER
var WhisperModel string

func GetWhisperModel() string {
	return WhisperModel
}

// =================================
// Set up Twilio configuration
func setUpTwilioConfig() {
	accountSid := os.Getenv("TWILIO_ACCOUNT_SID")
	if accountSid == "" {
		logger.LogErrorFatal("CONFIG", "Environment variable TWILIO_ACCOUNT_SID is unset")
	}

	TwilioConfig.AuthConfig.AccountSid = accountSid
	TwilioConfig.AuthConfig.AuthToken = os.Getenv("TWILIO_AUTH_TOKEN")
	TwilioConfig.AuthConfig.ApiKey = os.Getenv("TWILIO_API_KEY")
	TwilioConfig.AuthConfig.ApiSecret = os.Getenv("TWILIO_API_SECRET")

	if TwilioConfig.AuthConfig.ApiKey == "" || TwilioConfig.AuthConfig.ApiSecret == "" {
		logger.LogErrorFatal("CONFIG", "Twilio API credentials are (partly) missing in environment variables")
	}

	var defaultCallLength int = 38
	var callLength int
	s, exists := os.LookupEnv("TWILIO_CALL_LENGTH")
	if exists {
		c, err := strconv.Atoi(s)
		if err != nil {
			slog.Error("CONFIG", "message", "Failed converting TWILIO_CALL_LENGTH to int", "error", err)
			callLength = defaultCallLength
		} else {
			callLength = c
		}
	} else {
		callLength = defaultCallLength
	}

	slog.Info("CONFIG", "callLength", callLength, "envVarIsSet", exists, "usingDefaultValue", callLength == defaultCallLength)
	TwilioConfig.CallLength = callLength

	value := os.Getenv("TWILIO_CALL_FROM")
	if value == "" {
		logger.LogErrorFatal("CALLER", "TWILIO_CALL_FROM not set")
	}
	TwilioConfig.CallFrom = value

}

func init() {
	setUpTwilioConfig()
}
