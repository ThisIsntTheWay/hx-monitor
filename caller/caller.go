package caller

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/thisisnttheway/hx-checker/callback"
	"github.com/thisisnttheway/hx-checker/db"
	"github.com/thisisnttheway/hx-checker/logger"
	"github.com/thisisnttheway/hx-checker/models"
	"github.com/twilio/twilio-go"
	twilioApi "github.com/twilio/twilio-go/rest/api/v2010"
	"go.mongodb.org/mongo-driver/bson"
)

const twilioTimeFormat string = "Mon, 02 Jan 2006 15:04:05 -0700"

type CallResponse struct {
	SID         string
	Status      string
	Direction   string
	DateCreated time.Time
	EndTime     time.Time
	Price       float32
	PriceUnit   string
}

// Construct Twilio API client
func createTwilioClient() *twilio.RestClient {
	var twilioClientParams twilio.ClientParams
	accountSid := os.Getenv("TWILIO_ACCOUNT_SID")
	if accountSid == "" {
		logger.LogErrorFatal("CALLER", "Environment variable TWILIO_ACCOUNT_SID is unset")
	}

	authToken, exists := os.LookupEnv("TWILIO_AUTH_TOKEN")
	slog.Info("CALLER", "usingAuthToken", exists)
	if exists {
		twilioClientParams = twilio.ClientParams{
			Username: accountSid,
			Password: authToken,
		}
	} else {
		apiKey := os.Getenv("TWILIO_API_KEY")
		apiSecret := os.Getenv("TWILIO_API_SECRET")

		if apiKey == "" || apiSecret == "" {
			logger.LogErrorFatal("CALLER", "Twilio API credentials are (partly) missing in environment variables")
		} else {
			fmt.Printf(
				"Using the following credentials:\naccountSid: %s\napiKey: %s\napiSecret: %s\n",
				accountSid, apiKey, apiSecret,
			)
		}

		twilioClientParams = twilio.ClientParams{
			Username:   apiKey,
			Password:   apiSecret,
			AccountSid: accountSid,
		}
	}

	// Twilio region will be acquired by twilio-go by looking up TWILIO_REGION
	client := twilio.NewRestClientWithParams(twilioClientParams)

	return client
}

// Get numbers in database
func GetNumbers() []models.Number {
	results, err := db.GetDocument[models.Number]("numbers", bson.D{})
	if len(results) == 0 || err != nil {
		logger.LogErrorFatal("CALLER", "No numbers found")
	}

	slog.Info("CALLER", "message", fmt.Sprintf("Found %d number(s)", len(results)))
	return results
}

// Call a number and optionally start a live transcription
func Call(number string, startTranscription bool, startRecording bool) (CallResponse, error) {
	for {
		if !callback.IsCallbackurlSet() {
			slog.Warn("CALLER", "message", "Waiting for CallbackUrlDefined", "CallBackUrlDefined", callback.IsCallbackurlSet())
			time.Sleep(time.Second * 1)
		} else {
			break
		}
	}

	var callLength int
	var defaultCallLength int = 38
	s, exists := os.LookupEnv("TWILIO_CALL_LENGTH")
	if exists {
		c, err := strconv.Atoi(s)
		if err != nil {
			slog.Error("CALLER", "message", "Failed converting TWILIO_CALL_LENGTH to int", "error", err)
			callLength = defaultCallLength
		} else {
			callLength = c
		}
	} else {
		callLength = defaultCallLength
	}

	slog.Info("CALLER", "callLength", callLength, "envVarIsSet", exists, "usingDefaultValue", callLength == defaultCallLength)

	client := createTwilioClient()

	twilioCallFrom := os.Getenv("TWILIO_CALL_FROM")
	if twilioCallFrom == "" {
		logger.LogErrorFatal("CALLER", "TWILIO_CALL_FROM not set")
	}

	var targetNumber string = number
	if !strings.HasPrefix(number, "+41") {
		targetNumber = fmt.Sprintf("+41%s", number)
	}

	params := &twilioApi.CreateCallParams{}
	params.SetTo(targetNumber)
	params.SetFrom(twilioCallFrom)
	params.SetTimeLimit(callLength + 5) // Ensures transcripts can complete
	params.SetStatusCallback(callback.GetStatusCallbackurl() + callback.UrlConfigs.Calls)
	params.SetStatusCallbackEvent([]string{"initiated", "answered", "completed"})

	if startTranscription && startRecording {
		slog.Warn("CALL", "message", "Both live transcription and call recording are enabled")
	}

	var additionalMl string
	if startTranscription {
		// Apparently you could use twilio-go/twiml/twiml.go instead of assembling a string but idk how
		transcriptionHints := "$DAY, CTR, TMA, active, inactive"

		additionalParams := fmt.Sprintf("partialResults='%v' track='inbound_track'", callback.UsesPartialTranscriptionResults())
		additionalMl = fmt.Sprintf(
			"<Start><Transcription hints='%s' statusCallbackUrl='%s' %s/></Start>",
			transcriptionHints, callback.GetStatusCallbackurl()+callback.UrlConfigs.Transcriptions,
			additionalParams,
		)
	}

	if startRecording {
		additionalMl = fmt.Sprintf(
			"<Record maxLength='%d' playBeep='%v' recordingStatusCallback='%s'/>",
			callLength, false, callback.GetStatusCallbackurl()+callback.UrlConfigs.Recordings,
		)
	}

	slog.Info("CALLER", "action", "addAdditionalMl", "value", additionalMl)
	twiMl := fmt.Sprintf(
		"<Response>%s<Pause length='%d'/></Response>",
		additionalMl,
		callLength,
	)
	params.SetTwiml(twiMl)

	resp, err := client.Api.CreateCall(params)
	if err != nil {
		slog.Error("CALLER", "error", fmt.Sprintf("Error calling %s: %v", targetNumber, err.Error()))
		return CallResponse{}, err
	} else {
		var err error
		var parsedTime time.Time
		if resp.DateCreated != nil {
			timeString := *resp.DateCreated
			parsedTime, err = time.Parse(twilioTimeFormat, timeString)
			if err != nil {
				slog.Error("CALLER", "message", "Failed parsing reported DateCreated", "source", timeString, "error", err.Error())
				parsedTime = time.Now()
			}
		}

		var price float64
		if resp.Price != nil {
			price, err = strconv.ParseFloat(*resp.Price, 32)
			if err != nil {
				slog.Error("CALLER", "message", "Failed converting reported price", "source", *resp.Price, "error", err.Error())
				price = 0
			}
		}

		returnObj := CallResponse{
			Status:      *resp.Status,
			SID:         *resp.Sid,
			Direction:   *resp.Direction,
			DateCreated: parsedTime,
			Price:       float32(price),
			PriceUnit:   *resp.PriceUnit,
		}

		// Check the API for immediate errors
		time.Sleep(time.Second * 5)
		callDetails, err := client.Api.FetchCall(*resp.Sid, nil)
		if err != nil {
			slog.Error("CALLER", "message", "Failed fetching call", "sid", *resp.Sid)
			return CallResponse{}, err
		} else if callDetails.Status != nil && *callDetails.Status == "failed" {
			return CallResponse{}, fmt.Errorf("Call failed with status '%s'", *callDetails.Status)
		}

		slog.Info("CALLER", "message", fmt.Sprintf("Success calling %s", targetNumber), "call", returnObj)
		return returnObj, nil
	}
}

// Check a call for a given call SID
func CheckCall(sid string) (CallResponse, error) {
	client := createTwilioClient()

	params := &twilioApi.FetchCallParams{}
	resp, err := client.Api.FetchCall(sid, params)
	if err != nil {
		slog.Error("CALLER", "action", "fetch", "sid", sid, "error", err.Error())
		return CallResponse{}, err
	} else {
		timeCreatedString := *resp.DateCreated
		timeEndedString := *resp.EndTime
		parsedCreatedTime, err := time.Parse(twilioTimeFormat, timeCreatedString)
		if err != nil {
			slog.Error("CALLER", "message", "Failed parsing reported DateCreated", "source", timeCreatedString, "error", err.Error())
			parsedCreatedTime = time.Now()
		}

		parsedEndedTime, err := time.Parse(twilioTimeFormat, timeEndedString)
		if err != nil {
			slog.Error("CALLER", "message", "Failed parsing reported DateCreated", "source", timeEndedString, "error", err.Error())
			parsedEndedTime = time.Now()
		}

		price, err := strconv.ParseFloat(*resp.Price, 32)
		if err != nil {
			slog.Error("CALLER", "message", "Failed converting reported price", "source", *resp.Price, "error", err.Error())
			price = 0
		}

		returnObj := CallResponse{
			Status:      *resp.Status,
			SID:         *resp.Sid,
			Direction:   *resp.Direction,
			DateCreated: parsedCreatedTime,
			Price:       float32(price),
			PriceUnit:   *resp.PriceUnit,
			EndTime:     parsedEndedTime,
		}

		slog.Info("CALLER", "action", "fetch", "sid", sid, "response", returnObj)
		return returnObj, nil
	}
}

// Deletes a recording
func DeleteRecording(sid string) error {
	client := createTwilioClient()
	params := &twilioApi.DeleteRecordingParams{}

	return client.Api.DeleteRecording(sid, params)
}
