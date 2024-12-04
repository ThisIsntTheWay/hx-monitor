package main

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ggerganov/whisper.cpp/bindings/go/pkg/whisper"
	wav "github.com/go-audio/wav"
	"github.com/thisisnttheway/hx-checker/logger"
)

// File path of whisper model
var _whisperModel string

const whisperModelsFilePath = "./models"

func init() {
	for _, program := range []string{"ffmpeg", "ffprobe"} {
		cmd := exec.Command(program, "-version")
		err := cmd.Run()
		if err != nil {
			logger.LogErrorFatal("WHISPER", fmt.Sprintf("'%s' is not installed", program))
		}
	}

	whisperModel, exists := os.LookupEnv("WHISPER_MODEL")
	if !exists {
		logger.LogErrorFatal("WHISPER", "WHISPER_MODEL is unset")
	}
	_whisperModel, _ = getWhisperModel(whisperModel)
}

func downloadModelFromHuggingFace(model string) error {
	outputFilePath := filepath.Join(whisperModelsFilePath, model)
	url := fmt.Sprintf("https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-%s?download=true", model)

	fmt.Printf("[i] Downloading model: %s\n", url)

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Download failed: %s", resp.Status)
	}

	outFile, err := os.Create(outputFilePath)
	if err != nil {
		return fmt.Errorf("Creating file failed: %v", err)
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		return fmt.Errorf("Saving file failed: %v", err)
	}

	return nil
}

// Returns the file path of a model, download it first if missing
func getWhisperModel(model string) (string, error) {
	// Append .bin suffix
	fileNameSplit := strings.Split(model, ".")
	suffix := fileNameSplit[len(fileNameSplit)-1]
	if strings.ToLower(suffix) != "bin" {
		model = model + ".bin"
	}

	modelFilePath := filepath.Join(whisperModelsFilePath, model)
	if _, err := os.Stat(modelFilePath); errors.Is(err, os.ErrNotExist) {
		slog.Warn("WHISPER",
			"action", "getWhisperModel",
			"model", model,
			"doesNotExist", true,
		)

		// Attempt download, only supported if model is hosted on huggingface
		slog.Info("WHISPER", "action", "downloadWhisperModel", "model", model)
		err := downloadModelFromHuggingFace(model)
		if err != nil {
			logger.LogErrorFatal("WHISPER", fmt.Sprintf("Failed downloading model: %v", err))
		}
	}

	return modelFilePath, nil
}

// Transcribes a WAV file and returns the transcription
func Transcribe(filePath string) (string, error) {
	var transcript string
	compatibleFilepath, err := ensureRecordingIsCompatible(filePath)
	if err != nil {
		return "", err
	}

	// Decode WAV file as we'll need it as a float32
	fmt.Printf("[i] Will open file: %s\n", compatibleFilepath)
	file, err := os.Open(compatibleFilepath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var data []float32
	dec := wav.NewDecoder(file)
	if buf, err := dec.FullPCMBuffer(); err != nil {
		return "", err
	} else {
		data = buf.AsFloat32Buffer().Data
	}

	// Transcribe
	model, err := whisper.New(_whisperModel)
	if err != nil {
		logger.LogErrorFatal("WHISPER", fmt.Sprintf("whisper.New() err: %v", err))
	}
	defer model.Close()

	ctx, err := model.NewContext()
	if err != nil {
		logger.LogErrorFatal("WHISPER", fmt.Sprintf("NewContext() err: %v", err))
	}

	ctx.SetLanguage("en")
	fmt.Printf("[i] Whisper ctx language: %s\n", ctx.Language())
	if err := ctx.Process(data, nil, nil); err != nil {
		return "", err
	}

	for {
		segment, err := ctx.NextSegment()
		if err != nil {
			break
		}
		slog.Info("WHISPER", "action", "ctxNextSegment", "text", segment.Text)
		transcript = transcript + " " + segment.Text
	}

	return transcript, nil
}