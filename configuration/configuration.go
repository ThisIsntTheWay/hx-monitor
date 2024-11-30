package configuration

// --------------------------
// CALLBACK
type UrlConfig struct {
	Calls          string
	Transcriptions string
	Recordings     string
}

var UrlConfigs UrlConfig = UrlConfig{
	Calls:          "/calls",
	Transcriptions: "/transcription",
	Recordings:     "/recording",
}

var CallbackUrl string

func SetCallbackUrl(value string) {
	CallbackUrl = value
}

func GetCallbackUrl() string {
	return CallbackUrl
}

func IsCallbackurlSet() bool {
	return CallbackUrl != ""
}

// --------------------------
// TWILIO
var UsePartialTranscriptionResults bool

func UsesPartialTranscriptionResults() bool {
	return UsePartialTranscriptionResults
}

func SetPartialTranscriptionResultBool(value bool) {
	UsePartialTranscriptionResults = value
}
