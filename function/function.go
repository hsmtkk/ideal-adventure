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
	"google.golang.org/api/idtoken"
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
	var storageData storageObjectData
	if err := e.DataAs(&storageData); err != nil {
		return fmt.Errorf("event.Event.DataAs failed; %w", err)
	}
	imageBytes, err := getImage(ctx, storageData.Bucket, storageData.Name)
	if err != nil {
		return err
	}
	if err := callEndpoint(ctx, projectID, endpointID, imageBytes); err != nil {
		return err
	}
	return nil
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

func callEndpoint(ctx context.Context, projectID, endpointID string, imageBytes []byte) error {
	log.Println("callEndpoint")
	audience := "https://us-central1-aiplatform.googleapis.com"
	targetURL := fmt.Sprintf("https://us-central1-aiplatform.googleapis.com/v1/projects/%s/locations/us-central1/endpoints/%s:predict", projectID, endpointID)
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
	clt, err := idtoken.NewClient(ctx, audience)
	if err != nil {
		return fmt.Errorf("idtoken.NewClient failed; %w", err)
	}
	resp, err := clt.Post(targetURL, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("http.Client.Post failed; %w", err)
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
	return nil
}
