package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"google.golang.org/api/idtoken"
)

type Invoker interface {
	Invoke(imageBytes []byte) (string, error)
}

func New(projectID, endPointID string) Invoker {
	return &invokerImpl{projectID, endPointID}
}

type invokerImpl struct {
	projectID  string
	endPointID string
}

/*
{
  "instances": [{
    "content": "YOUR_IMAGE_BYTES"
  }],
  "parameters": {
    "confidenceThreshold": 0.5,
    "maxPredictions": 5
  }
}
*/

type schema struct {
	Instances  []instance `json:"instances"`
	Parameters parameters `json:"parameters"`
}

type instance struct {
	Content string `json:"content"`
}

type parameters struct {
	ConfidenceThreshold float64 `json:"confidenceThreshold"`
	MaxPredictions      int     `json:"maxPredictions"`
}

func (i *invokerImpl) Invoke(imageBytes []byte) (string, error) {
	ctx := context.Background()
	audience := "https://us-central1-aiplatform.googleapis.com"
	targetURL := fmt.Sprintf("https://us-central1-aiplatform.googleapis.com/v1/projects/%s/locations/us-central1/endpoints/%s:predict", i.projectID, i.endPointID)

	reqBody, err := json.Marshal(schema{
		Instances: []instance{},
		Parameters: parameters{
			ConfidenceThreshold: 0.5,
			MaxPredictions:      5,
		},
	})
	if err != nil {
		return "", fmt.Errorf("json.Marshal failed; %w", err)
	}

	client, err := idtoken.NewClient(ctx, audience)
	if err != nil {
		return "", fmt.Errorf("idtoken.NewClient failed; %w", err)
	}

	resp, err := client.Post(targetURL, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("http.Client.Post failed; %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("non 200 HTTP status code; %d; %s", resp.StatusCode, resp.Status)
	}

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("io.ReadAll failed; %w", err)
	}

	log.Printf(string(respBytes))
	return "", nil
}
