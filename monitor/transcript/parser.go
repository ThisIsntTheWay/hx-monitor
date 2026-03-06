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

	o, _ := json.Marshal(areaMeiringenStatus)
	slog.Debug("PARSER", "airspaceStatusJson", string(o))

	return areaMeiringenStatus, nil
}
