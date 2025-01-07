package transcript

import (
	"bytes"
	"log/slog"
	"os/exec"
	"regexp"
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

	sanitizedOutput := regexp.MustCompile(`\d+`).FindString(out.String())
	sampleRate, err := strconv.Atoi(sanitizedOutput)
	slog.Info("AUDIO", "action", "checkSampleRate", "reference", filePath, "result", sampleRate, "error", err)

	return sampleRate, err
}

// Convert file to WAV format, returns filePath of converted file
func convertToWav(filePath string) (string, error) {
	s := strings.Split(filePath, ".")
	suffix := s[len(s)-1]
	outputFile := strings.TrimSuffix(filePath, "."+suffix) + "_converted.wav"
	slog.Info("AUDIO", "action", "convertToWav", "outputFile", outputFile)

	var out bytes.Buffer
	cmd := exec.Command("ffmpeg", "-y", "-i", filePath, "-ar", "16000", "-ac", "1", outputFile)
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		slog.Error("AUDIO", "exec", "ffmpeg", "stderr", out.String())
		return "", err
	}

	return outputFile, nil
}

// Convert WAV file to 16kHz, returns filePath of converted file
func convertTo16kHz(filePath string) (string, error) {
	s := strings.Split(filePath, ".")
	suffix := s[len(s)-1]
	outputFile := strings.TrimSuffix(filePath, "."+suffix) + "_16kHz.wav"
	slog.Info("AUDIO", "action", "convertTo16kHz", "outputFile", outputFile)

	cmd := exec.Command("ffmpeg", "-y", "-i", filePath, "-ar", "16000", outputFile)
	err := cmd.Run()
	if err != nil {
		return "", err
	}

	return outputFile, nil
}

// Converts a recording into an expected format, if applicable. Returns absolute file path of guaranteed compatible sample.
func ensureRecordingIsCompatible(filePath string) (string, error) {
	var expectedSampleRate int = 16000
	returnFilePath := filePath

	split := strings.Split(filePath, ".")
	if strings.ToLower(split[len(split)-1]) != "wav" {
		var err error
		returnFilePath, err = convertToWav(filePath)
		if err != nil {
			slog.Error("WHISPER",
				"action", "convertToWav",
				"filePath", filePath,
				"error", err,
			)

			return "", err
		}
	}

	sampleRate, err := checkSampleRate(returnFilePath)
	if err != nil {
		slog.Error("WHISPER",
			"action", "checkSampleRate",
			"filePath", returnFilePath,
			"error", err,
		)
		return "", err
	}

	if sampleRate != expectedSampleRate {
		returnFilePath, err := convertTo16kHz(returnFilePath)
		if err != nil {
			slog.Error("WHISPER",
				"action", "convertTo16kHz",
				"filePath", returnFilePath,
				"error", err,
			)
		}
		return "", err
	}

	return returnFilePath, nil
}
