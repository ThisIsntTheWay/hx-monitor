package caller

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/thisisnttheway/hx-checker/callback"
	"github.com/thisisnttheway/hx-checker/db"
	"github.com/thisisnttheway/hx-checker/logger"
	"github.com/thisisnttheway/hx-checker/models"
	"github.com/twilio/twilio-go"
	twilioApi "github.com/twilio/twilio-go/rest/api/v2010"
	"go.mongodb.org/mongo-driver/bson"
)

var twilioTimeFormat string = "Mon, 02 Jan 2006 15:04:05 -0700"

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
func constructClient() *twilio.RestClient {
	accountSid := os.Getenv("TWILIO_ACCOUNT_SID")
	apiKey := os.Getenv("TWILIO_API_KEY")
	apiSecret := os.Getenv("TWILIO_API_SECRET")
	// Twilio region will be acquired by twilio-go by looking up TWILIO_REGION

	if accountSid == "" || apiKey == "" || apiSecret == "" {
		logger.LogErrorFatal("CALLER", "Twilio API credentials are (partly) missing in environment variables")
	}

	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username:   apiKey,
		Password:   apiSecret,
		AccountSid: accountSid,
	})

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
func Call(number string, startTranscription bool) (CallResponse, error) {
	client := constructClient()

	twilioCallFrom := os.Getenv("TWILIO_CALL_FROM")
	if twilioCallFrom == "" {
		logger.LogErrorFatal("CALLER", "TWILIO_CALL_FROM not set")
	}

	targetNumber := fmt.Sprintf("+41%s", number)

	params := &twilioApi.CreateCallParams{}
	params.SetTo(targetNumber)
	params.SetFrom(twilioCallFrom)
	params.SetTimeout(10)
	params.SetTimeLimit(30)
	params.SetStatusCallback(callback.CallbackUrl + "/call")
	params.SetStatusCallbackEvent([]string{"initiated", "answered", "completed"})

	if startTranscription {
		transcriptionHints := "active,inactive,Meiringen,CTR,TMA"
		twiMl := fmt.Sprintf(
			"<Response><Start><Transcription hints='%s' statusCallbackUrl='%s'/></Start><Pause length='30'/></Response>",
			transcriptionHints,
			callback.CallbackUrl+"/transcription",
		)
		params.SetTwiml(twiMl)
	}

	resp, err := client.Api.CreateCall(params)
	if err != nil {
		slog.Error("CALLER", "error", fmt.Sprintf("Error calling %s: %v", targetNumber, err.Error()))
		return CallResponse{}, err
	} else {
		timeString := *resp.DateCreated
		parsedTime, err := time.Parse(twilioTimeFormat, timeString)
		if err != nil {
			slog.Error("CALLER", "message", "Failed parsing reported DateCreated", "source", timeString, "error", err.Error())
			parsedTime = time.Now()
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
			DateCreated: parsedTime,
			Price:       float32(price),
			PriceUnit:   *resp.PriceUnit,
		}

		slog.Info("CALLER", "message", fmt.Sprintf("Success calling %s: %T", targetNumber, returnObj))
		return returnObj, nil
	}
}

// Check a call for a given call SID
func CheckCall(sid string) (CallResponse, error) {
	client := constructClient()

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
