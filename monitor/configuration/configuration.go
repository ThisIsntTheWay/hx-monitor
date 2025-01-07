package configuration

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"github.com/thisisnttheway/hx-monitor/logger"
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

var twilioConfig TwilioConfiguration

func UsesPartialTranscriptionResults() bool {
	return twilioConfig.UsePartialTranscriptionResults
}

func SetPartialTranscriptionResultBool(value bool) {
	twilioConfig.UsePartialTranscriptionResults = value
}

func GetTwilioConfig() TwilioConfiguration {
	return twilioConfig
}

func SetUpTwilioConfig() {
	accountSid := os.Getenv("TWILIO_ACCOUNT_SID")
	if accountSid == "" {
		logger.LogErrorFatal("CONFIG", "Environment variable TWILIO_ACCOUNT_SID is unset")
	}

	twilioConfig.AuthConfig.AccountSid = accountSid
	twilioConfig.AuthConfig.AuthToken = os.Getenv("TWILIO_AUTH_TOKEN")
	twilioConfig.AuthConfig.ApiKey = os.Getenv("TWILIO_API_KEY")
	twilioConfig.AuthConfig.ApiSecret = os.Getenv("TWILIO_API_SECRET")

	if twilioConfig.AuthConfig.ApiKey == "" || twilioConfig.AuthConfig.ApiSecret == "" {
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
	twilioConfig.CallLength = callLength

	value := os.Getenv("TWILIO_CALL_FROM")
	if value == "" {
		logger.LogErrorFatal("CALLER", "TWILIO_CALL_FROM not set")
	}
	twilioConfig.CallFrom = value
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

var mongoConfig MongoConfiguration

func GetMongoConfig() MongoConfiguration {
	return mongoConfig
}

// Set up MongoDB configuration
func SetUpMongoConfig() {
	mongoConfig.Database = getEnv("MONGODB_DATABASE", "hx")
	mongoConfig.Username = getEnv("MONGO_USER", "")
	mongoConfig.Password = getEnv("MONGO_PASSWORD", "")
	mongoConfig.Host = getEnv("MONGO_HOST", "")
	mongoConfig.Port = getEnv("MONGO_PORT", "")

	if mongoConfig.Host == "" || mongoConfig.Port == "" {
		logger.LogErrorFatal("DB", "MongoDB connection details are missing in environment variables")
	}
	if mongoConfig.Username == "" || mongoConfig.Password == "" {
		logger.LogErrorFatal("DB", "MongoDB connection credentials are missing in environment variables")
	}

	mongoConfig.Uri = fmt.Sprintf(
		"mongodb://%s:%s@%s:%s",
		mongoConfig.Username,
		mongoConfig.Password,
		mongoConfig.Host,
		mongoConfig.Port,
	)
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
