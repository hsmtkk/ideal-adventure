package helloworld

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/cloudevents/sdk-go/v2/event"
)

func init() {
	functions.CloudEvent("imageUploaded", imageUploaded)
}

// https://cloud.google.com/functions/docs/samples/functions-cloudevent-storage?hl=ja

type storageObjectData struct {
	Bucket         string    `json:"bucket,omitempty"`
	Name           string    `json:"name,omitempty"`
	Metageneration int64     `json:"metageneration,string,omitempty"`
	TimeCreated    time.Time `json:"timeCreated,omitempty"`
	Updated        time.Time `json:"updated,omitempty"`
}

func imageUploaded(ctx context.Context, e event.Event) error {
	projectID := os.Getenv("PROJECT_ID")
	if projectID == "" {
		return fmt.Errorf("PROJECT_ID env var is not defined")
	}
	endpointID := os.Getenv("ENDPOINT_ID")
	if endpointID == "" {
		return fmt.Errorf("ENDPOINT_ID env var is not defined")
	}
	token, err := getToken()
	if err != nil {
		log.Fatal(err)
	}
	var storageData storageObjectData
	if err := e.DataAs(&storageData); err != nil {
		return fmt.Errorf("event.Event.DataAs failed; %w", err)
	}
	imageBytes, err := getImage(ctx, storageData.Bucket, storageData.Name)
	if err != nil {
		return err
	}
	if err := callEndpoint(ctx, projectID, endpointID, token, imageBytes); err != nil {
		return err
	}
	return nil
}

/*
https://cloud.google.com/functions/docs/securing/function-identity
curl "http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/token?scopes=SCOPES" \
  -H "Metadata-Flavor: Google"
*/

type tokenSchema struct {
	AccessToken string `json:"access_token"`
}

func getToken() (string, error) {
	url := fmt.Sprintf("http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/token?scopes=%s", "https://www.googleapis.com/auth/cloud-platform")
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("http.NewRequest failed; %w", err)
	}
	req.Header.Add("Metadata-Flavor", "Google")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("http.Client.Do failed; %w", err)
	}
	defer resp.Body.Close()
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("io.ReadAll failed; %w", err)
	}
	var token tokenSchema
	if err := json.Unmarshal(respBytes, &token); err != nil {
		return "", fmt.Errorf("json.Unmarshal failed; %w", err)
	}
	return token.AccessToken, nil
}

func getImage(ctx context.Context, bucket, name string) ([]byte, error) {
	log.Println("getImage")
	clt, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("storage.NewClient failed; %w", err)
	}
	defer clt.Close()
	var buf bytes.Buffer
	reader, err := clt.Bucket(bucket).Object(name).NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("storage.ObjectHandle.NewReader failed; %w", err)
	}
	defer reader.Close()
	if _, err := io.Copy(&buf, reader); err != nil {
		return nil, fmt.Errorf("io.Copy failed; %w", err)
	}
	return buf.Bytes(), nil
}

type requestSchema struct {
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

/*
response sample

{
  "predictions": [
    {
      "confidences": [
        0.999999285
      ],
      "ids": [
        "5378020883975634944"
      ],
      "displayNames": [
        "apple"
      ]
    }
  ],
  "deployedModelId": "2683525253354749952",
  "model": "projects/864499401284/locations/us-central1/models/116743945614000128",
  "modelDisplayName": "untitled_1671069571418",
  "modelVersionId": "1"
}
*/

type responseSchema struct {
	Predictions []prediction `json:"predictions"`
}

type prediction struct {
	DisplayNames []string `json:"displayNames"`
}

func callEndpoint(ctx context.Context, projectID, endpointID, token string, imageBytes []byte) error {
	log.Println("callEndpoint")
	url := fmt.Sprintf("https://us-central1-aiplatform.googleapis.com/v1/projects/%s/locations/us-central1/endpoints/%s:predict", projectID, endpointID)
	reqBody, err := json.Marshal(requestSchema{
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
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("http.NewRequest failed; %w", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("http.Client.Do failed; %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("non 200 HTTP status code; %d; %s", resp.StatusCode, resp.Status)
	}
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("io.ReadAll failed; %w", err)
	}
	log.Printf("%v", string(respBytes))
	var apiResponse responseSchema
	if err := json.Unmarshal(respBytes, &apiResponse); err != nil {
		return fmt.Errorf("json.Unmarshal failed; %w", err)
	}
	log.Printf("result: %s\n", apiResponse.Predictions[0].DisplayNames[0])
	return nil
}
