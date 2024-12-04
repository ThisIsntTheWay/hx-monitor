package main

import (
	"bytes"
	"fmt"
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
	return strconv.Atoi(sanitizedOutput)
}

// Convert file to WAV format, returns filePath of converted file
func convertToWav(filePath string) (string, error) {
	s := strings.Split(filePath, ".")
	suffix := s[len(s)-1]
	outputFile := strings.TrimSuffix(filePath, "."+suffix) + "_converted.wav"

	cmd := exec.Command("ffmpeg", "-y", "-i", filePath, "-ar", "16000", "-ac", "1", outputFile)

	var out bytes.Buffer
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		fmt.Printf("[X] stderr: %v\n", out.String())
		fmt.Printf("[X] err: %v\n", err)
		return "", err
	}

	return outputFile, nil
}

// Convert WAV file to 16kHz, returns filePath of converted file
func convertTo16kHz(filePath string) (string, error) {
	s := strings.Split(filePath, ".")
	suffix := s[len(s)-1]
	outputFile := strings.TrimSuffix(filePath, "."+suffix) + "_16kHz.wav"

	cmd := exec.Command("ffmpeg", "-i", filePath, "-ar", "16000", outputFile)
	err := cmd.Run()
	if err != nil {
		return "", err
	}

	return outputFile, nil
}

// Converts a recording into an expected format, if applicable. Returns absolute file path of guaranteed compatible sample.
func ensureRecordingIsCompatible(filePath string) (string, error) {
	fmt.Printf("[i] Ensuring file compatibility: %s\n", filePath)
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

	fmt.Printf("[1] convertToWav: %s\n", returnFilePath)

	sampleRate, err := checkSampleRate(returnFilePath)
	if err != nil {
		slog.Error("WHISPER",
			"action", "checkSampleRate",
			"filePath", returnFilePath,
			"error", err,
		)
		return "", err
	}

	fmt.Printf("[2] checkSampleRate: %s\n", returnFilePath)

	if sampleRate != expectedSampleRate {
		fmt.Println("[2.1] Will convert to 16Khz")
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

	fmt.Printf("[3] convertTo16kHz: %s\n", returnFilePath)

	return returnFilePath, nil
}
