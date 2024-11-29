package transcript

import (
	"bytes"
	"os/exec"
	"strconv"
	"strings"
)

// Check if a file is a WAV file and determine its sample rate
func checkSampleRate(filePath string) (int, error) {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-select_streams", "a",
		"-of", "default=noprint_wrappers=1:nokey=1",
		"-show_entries",
		"stream=sample_rate",
		filePath,
	)

	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return 0, err
	}

	return strconv.Atoi(out.String())
}

// Convert file to WAV format
func convertToWav(filePath string) (string, error) {
	outputFile := strings.TrimSuffix(filePath, ".wav") + "_converted.wav"
	cmd := exec.Command("ffmpeg", "-i", filePath, "-ar", "16000", "-ac", "1", outputFile)
	err := cmd.Run()
	if err != nil {
		return "", err
	}

	return outputFile, nil
}

// Convert WAV file to 16kHz
func convertTo16kHz(filePath string) (string, error) {
	outputFile := strings.TrimSuffix(filePath, ".wav") + "_16kHz.wav"
	cmd := exec.Command("ffmpeg", "-i", filePath, "-ar", "16000", outputFile)
	err := cmd.Run()
	if err != nil {
		return "", err
	}

	return outputFile, nil
}
