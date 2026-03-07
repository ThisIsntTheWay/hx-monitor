package caller

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	c "github.com/thisisnttheway/hx-monitor/configuration"
	"github.com/thisisnttheway/hx-monitor/db"
	"github.com/thisisnttheway/hx-monitor/logger"
	"github.com/thisisnttheway/hx-monitor/models"
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

	usesAuthToken := c.GetTwilioConfig().AuthConfig.AuthToken != ""
	slog.Info("CALLER", "action", "createClient", "usingAuthToken", usesAuthToken)
	if usesAuthToken {
		twilioClientParams = twilio.ClientParams{
			Username: c.GetTwilioConfig().AuthConfig.AccountSid,
			Password: c.GetTwilioConfig().AuthConfig.AuthToken,
		}
	} else {
		twilioClientParams = twilio.ClientParams{
			Username:   c.GetTwilioConfig().AuthConfig.ApiKey,
			Password:   c.GetTwilioConfig().AuthConfig.ApiSecret,
			AccountSid: c.GetTwilioConfig().AuthConfig.AccountSid,
		}
	}

	// Twilio region and edge will be acquired by twilio-go by looking up TWILIO_REGION & TWILIO_EDGE
	client := twilio.NewRestClientWithParams(twilioClientParams)
	slog.Info("CONFIG", "twilioRegion", client.Region, "twilioEdge", client.Edge)

	return client
}

// Get numbers in database
func GetNumbers() []models.Number {
	results, err := db.GetDocument[models.Number]("numbers", bson.D{})
	if len(results) == 0 || err != nil {
		logger.LogErrorFatal("CALLER", "No numbers found")
	}

	slog.Info("CALLER", "action", "getNumbers", "amount", len(results))
	return results
}

// Call a number and optionally start a live transcription
func Call(number string, startTranscription bool, startRecording bool) (CallResponse, error) {
	for {
		if !c.IsCallbackurlSet() {
			slog.Warn("CALLER", "message", "Waiting for CallbackUrlDefined", "CallBackUrlDefined", c.IsCallbackurlSet())
			time.Sleep(time.Second * 1)
		} else {
			break
		}
	}

	client := createTwilioClient()

	var targetNumber string = number
	if !strings.HasPrefix(number, "+41") {
		targetNumber = fmt.Sprintf("+41%s", number)
	}

	params := &twilioApi.CreateCallParams{}
	params.SetTo(targetNumber)
	params.SetFrom(c.GetTwilioConfig().CallFrom)
	params.SetTimeLimit(c.GetTwilioConfig().CallLength + 5) // Ensures transcripts can complete
	params.SetStatusCallback(c.GetCallbackUrl() + c.UrlConfigs.Calls)
	params.SetStatusCallbackEvent([]string{"initiated", "answered", "completed"})

	if startTranscription && startRecording {
		slog.Warn("CALLER", "message", "Both live transcription and call recording are enabled")
	}

	var additionalMl string
	if startTranscription {
		// Apparently you could use twilio-go/twiml/twiml.go instead of assembling a string but idk how
		transcriptionHints := "$DAY, CTR, TMA, active, inactive"

		additionalParams := fmt.Sprintf("partialResults='%v' track='inbound_track'", c.UsesPartialTranscriptionResults())
		additionalMl = fmt.Sprintf(
			"<Start><Transcription hints='%s' statusCallbackUrl='%s' %s/></Start>",
			transcriptionHints, c.GetCallbackUrl()+c.UrlConfigs.Transcriptions,
			additionalParams,
		)
	}

	if startRecording {
		additionalMl = fmt.Sprintf(
			"<Record maxLength='%d' playBeep='%v' recordingStatusCallback='%s'/>",
			c.GetTwilioConfig().CallLength, false, c.GetCallbackUrl()+c.UrlConfigs.Recordings,
		)
	}

	slog.Info("CALLER", "action", "addAdditionalMl", "value", additionalMl)
	twiMl := fmt.Sprintf(
		"<Response>%s<Pause length='%d'/></Response>",
		additionalMl,
		c.GetTwilioConfig().CallLength,
	)
	params.SetTwiml(twiMl)

	resp, err := client.Api.CreateCall(params)
	if err != nil {
		slog.Error("CALLER", "error", fmt.Sprintf("Error calling %s: %v", targetNumber, err.Error()))
		return CallResponse{}, err
	} else {
		var err error
		var parsedCreatedTime time.Time
		if resp.DateCreated != nil {
			timeString := *resp.DateCreated
			parsedCreatedTime, err = time.Parse(twilioTimeFormat, timeString)
			if err != nil {
				slog.Error("CALLER", "message", "Failed parsing reported DateCreated", "source", timeString, "error", err.Error())
				parsedCreatedTime = time.Now()
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

		// Safely extract values from resp to avoid nil pointer dereferences
		var status, sid, direction, priceUnit string
		if resp.Status != nil {
			status = *resp.Status
		}
		if resp.Sid != nil {
			sid = *resp.Sid
		}
		if resp.Direction != nil {
			direction = *resp.Direction
		}
		if resp.PriceUnit != nil {
			priceUnit = *resp.PriceUnit
		}

		// Parse EndTime safely
		var parsedEndedTime time.Time
		if resp.EndTime != nil {
			timeEndedString := *resp.EndTime
			if t, err := time.Parse(twilioTimeFormat, timeEndedString); err == nil {
				parsedEndedTime = t
			} else {
				slog.Error("CALLER", "message", "Failed parsing reported DateEnded", "source", timeEndedString, "error", err.Error())
				parsedEndedTime = time.Now()
			}
		} else {
			parsedEndedTime = time.Now()
		}

		returnObj := CallResponse{
			Status:      status,
			SID:         sid,
			Direction:   direction,
			DateCreated: parsedCreatedTime,
			Price:       float32(price),
			PriceUnit:   priceUnit,
			EndTime:     parsedEndedTime,
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
