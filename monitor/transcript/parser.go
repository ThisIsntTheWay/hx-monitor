package transcript

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	_ "embed"

	"github.com/thisisnttheway/hx-monitor/models"
	"google.golang.org/genai"
)

var (
	model       string  = "gemini-flash-lite-latest"
	temperature float32 = 0.1

	//go:embed sysprompt_meiringen.txt
	syspromptMeiringen string
)

func init() {
	v, exists := os.LookupEnv("GOOGLE_AI_MODEL")
	if exists {
		model = v
	}

	slog.Info("PARSER", "aiModelToUse", model, "fromEnvVar", exists)
}

// Parse Meiringens airspace status phone system
func ParseAirspaceTranscriptMeiringen(transcript string, ctx context.Context) (models.AirspaceMeiringenStatus, error) {
	areaMeiringenStatus := models.AirspaceMeiringenStatus{}

	syspromptMeiringen = strings.Replace(syspromptMeiringen, "%TIME%", time.Now().Format(time.RFC1123Z), 1)
	config := &genai.GenerateContentConfig{
		Temperature:      &temperature,
		ResponseMIMEType: "application/json",
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{
				{Text: syspromptMeiringen},
			},
		},
		//ResponseSchema: &genai.Schema{} - We can't pass models.AirspaceMeiringenStatus to it :(
		// However, the AI generally seems to respond with a correct schema
	}

	slog.Info("PARSER", "action", "startGeneration", "model", model, "input", transcript)
	result, err := genaiClient.Models.GenerateContent(
		ctx,
		model,
		genai.Text(transcript),
		config,
	)
	if err != nil {
		slog.Error("PARSER", "action", "startGeneration", "err", err)
		return areaMeiringenStatus, fmt.Errorf("could not generate content from AI: %v", err)
	}

	slog.Info("PARSER", "action", "receiveResponse",
		"text", result.Text(),
		"totalTokenCount", result.UsageMetadata.TotalTokenCount,
		"modelVersion", result.ModelVersion,
	)

	err = json.Unmarshal([]byte(result.Text()), &areaMeiringenStatus)
	if err != nil {
		slog.Error("PARSER", "action", "unmarshalGenAiContent", "err", err)
		return areaMeiringenStatus, fmt.Errorf("could not unmarshal AI response: %v", err)
	}

	// Ensure NextUpdate is in the swiss timezone
	loc, _ := time.LoadLocation("Europe/Zurich")
	areaMeiringenStatus.NextUpdate = areaMeiringenStatus.NextUpdate.In(loc)

	// Reprompt if nextUpdate is in the past (or now)
	now := time.Now()
	if !areaMeiringenStatus.NextUpdate.After(now) {
		slog.Warn("PARSER", "action", "nextUpdateInPast", "nextUpdate", areaMeiringenStatus.NextUpdate, "now", now)

		// Reprompt the model to reinterpret just the nextUpdate field
		repromptConfig := &genai.GenerateContentConfig{
			Temperature: &temperature,
			SystemInstruction: &genai.Content{
				Parts: []*genai.Part{
					{Text: "You are a Swiss airspace status parser. Determine when the next update will be from the following transcript. The current time is " + now.Format(time.RFC1123Z) + ""},
				},
			},
			ResponseMIMEType: "application/json",
			ResponseSchema: &genai.Schema{
				Type: "object",
				Properties: map[string]*genai.Schema{
					"nextUpdate": {
						Type:        "string",
						Description: "RFC3339 formatted timestamp",
					},
				},
				Required: []string{"nextUpdate"},
			},
		}

		slog.Info("PARSER", "action", "repromptForNextUpdate", "transcript", transcript)
		repromptResult, err := genaiClient.Models.GenerateContent(
			ctx,
			model,
			genai.Text(transcript),
			repromptConfig,
		)
		if err != nil {
			slog.Error("PARSER", "action", "repromptForNextUpdate", "err", err)
		} else {
			var nextUpdateData struct {
				NextUpdate time.Time `json:"nextUpdate"`
			}
			err = json.Unmarshal([]byte(repromptResult.Text()), &nextUpdateData)
			if err != nil {
				slog.Error("PARSER", "action", "unmarshalNextUpdateReprompt", "err", err)
			} else {
				areaMeiringenStatus.NextUpdate = nextUpdateData.NextUpdate.In(loc)
				slog.Info("PARSER", "action", "nextUpdateRepromptSucceeded", "newNextUpdate", areaMeiringenStatus.NextUpdate)
			}
		}
	}

	o, _ := json.Marshal(areaMeiringenStatus)
	slog.Debug("PARSER", "airspaceStatusJson", string(o))

	return areaMeiringenStatus, nil
}
