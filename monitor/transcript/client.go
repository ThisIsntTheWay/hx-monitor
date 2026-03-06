package transcript

import (
	"context"
	"log"

	"google.golang.org/genai"
)

var (
	genaiClient *genai.Client
)

func init() {
	ctx := context.Background()
	var err error
	genaiClient, err = genai.NewClient(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
}
