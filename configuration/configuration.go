package configuration

import (
	"fmt"
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
// DATABASE
type MongoConfiguration struct {
	Database string
	Username string
	Password string
	Host     string
	Port     string
	Uri      string
}

var MongoConfig MongoConfiguration

// Set up MongoDB configuration
func SetUpMongoConfig() {
	MongoConfig.Database = getEnv("MONGODB_DATABASE", "hx")
	MongoConfig.Username = getEnv("MONGO_USER", "")
	MongoConfig.Password = getEnv("MONGO_PASSWORD", "")
	MongoConfig.Host = getEnv("MONGO_HOST", "")
	MongoConfig.Port = getEnv("MONGO_PORT", "")

	if MongoConfig.Host == "" || MongoConfig.Port == "" {
		logger.LogErrorFatal("DB", "MongoDB connection details are missing in environment variables")
	}
	if MongoConfig.Username == "" || MongoConfig.Password == "" {
		logger.LogErrorFatal("DB", "MongoDB connection credentials are missing in environment variables")
	}

	MongoConfig.Uri = fmt.Sprintf(
		"mongodb://%s:%s@%s:%s",
		MongoConfig.Username,
		MongoConfig.Password,
		MongoConfig.Host,
		MongoConfig.Port,
	)
}

// =================================
// Set up Twilio configuration
func SetUpTwilioConfig() {
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

// =================================
// Get environment variable with a default value
func getEnv(key string, defaultValue string) string {
	val, ok := os.LookupEnv(key)
	if ok {
		return val
	} else {
		return defaultValue
	}
}
