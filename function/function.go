package helloworld

import (
	"context"
	"log"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/cloudevents/sdk-go/v2/event"
)

func init() {
	functions.CloudEvent("imageUploaded", imageUploaded)
}

func imageUploaded(ctx context.Context, e event.Event) error {
	log.Printf("%v", e)
	return nil
}
