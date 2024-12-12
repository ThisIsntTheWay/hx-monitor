package callback

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	c "github.com/thisisnttheway/hx-monitor/configuration"
)

// Downloads a recording and returns the absolute file path of the saved recording
func DownloadRecording(sid string, url string) (string, error) {
	const format string = ".mp3"
	recordingFileName := sid + format
	slog.Info("CALLBACK",
		"action", "downloadRecording",
		"url", url,
		"format", format,
	)

	req, _ := http.NewRequest("GET", url+format, nil)
	twilioAuthDetails := c.GetTwilioConfig().AuthConfig
	if twilioAuthDetails.AuthToken != "" {
		req.SetBasicAuth(
			twilioAuthDetails.AccountSid,
			twilioAuthDetails.AuthToken,
		)
	} else {
		req.SetBasicAuth(
			twilioAuthDetails.ApiKey,
			twilioAuthDetails.ApiSecret,
		)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("http status code is %d: %v", resp.StatusCode, resp.Body)
	}

	filePath := filepath.Join(os.TempDir(), recordingFileName)
	file, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return "", err
	}

	return filePath, nil
}
