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
	model string = "gemini-3-flash-preview"

	//go:embed sysprompt_meiringen.txt
	syspromptMeiringen string
)

func init() {
	if v, exists := os.LookupEnv("GOOGLE_AI_MODEL"); exists {
		model = v
	}
}

// Parse Meiringens airspace status phone system
func ParseAirspaceTranscriptMeiringen(transcript string, ctx context.Context) (models.AirspaceMeiringenStatus, error) {
	areaMeiringenStatus := models.AirspaceMeiringenStatus{}

	syspromptMeiringen = strings.Replace(syspromptMeiringen, "%TIME%", time.Now().Format("15:04:05"), 1)
	config := &genai.GenerateContentConfig{
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{
				{Text: syspromptMeiringen},
			},
		},
	}

	slog.Info("PARSER", "action", "startGeneration", "model", model, "input", transcript)
	result, err := genaiClient.Models.GenerateContent(
		ctx,
		model,
		genai.Text(transcript),
		config,
	)
	if err != nil {
		return areaMeiringenStatus, fmt.Errorf("could not generate content from AI: %v", err)
	}

	slog.Info("PARSER", "action", "receiveResponse",
		"text", result.Text(),
		"totalTokenCount", result.UsageMetadata.TotalTokenCount,
		"modelVersion", result.ModelVersion,
	)

	err = json.Unmarshal([]byte(result.Text()), &areaMeiringenStatus)
	if err != nil {
		return areaMeiringenStatus, fmt.Errorf("could not unmarshal AI response: %v", err)
	}

	return areaMeiringenStatus, nil
}
