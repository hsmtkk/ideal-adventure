package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"os"

	"github.com/spf13/cobra"
)

// gcloud auth print-access-token

func main() {
	cmd := &cobra.Command{
		Args: cobra.ExactArgs(4),
		Run:  run,
		Use:  "testapi projectID endPointID image accessToken",
	}
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func run(cmd *cobra.Command, args []string) {
	projectID := args[0]
	endPointID := args[1]
	image := args[2]
	accessToken := args[3]

	imageBytes, err := os.ReadFile(image)
	if err != nil {
		log.Fatal(err)
	}

	if err := callAPI(projectID, endPointID, imageBytes, accessToken); err != nil {
		log.Fatal(err)
	}
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

func callAPI(projectID, endPointID string, imageBytes []byte, accessToken string) error {
	targetURL := fmt.Sprintf("https://us-central1-aiplatform.googleapis.com/v1/projects/%s/locations/us-central1/endpoints/%s:predict", projectID, endPointID)

	reqBody, err := json.Marshal(schema{
		Instances: []instance{{
			Content: base64.StdEncoding.EncodeToString(imageBytes),
		}},
		Parameters: parameters{
			ConfidenceThreshold: 0.5,
			MaxPredictions:      5,
		},
	})
	if err != nil {
		return fmt.Errorf("json.Marshal failed; %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, targetURL, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("http.NewRequest failed; %w", err)
	}
	req.Header.Add("Authorization", "Bearer "+accessToken)
	req.Header.Add("Content-Type", "application/json")

	reqBytes, err := httputil.DumpRequest(req, true)
	if err != nil {
		return fmt.Errorf("httputil.DumpRequest failed; %w", err)
	}
	log.Println(string(reqBytes))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("http.Client.Do failed; %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("non 200 HTTP status code; %d; %s", resp.StatusCode, resp.Status)
	}

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("io.ReadAll failed; %w", err)
	}

	log.Println(string(respBytes))
	return nil
}
