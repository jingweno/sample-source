package main

import (
	"context"
	"log"

	"github.com/cloudevents/sdk-go/pkg/cloudevents"
	"github.com/knative/eventing-sources/pkg/kncloudevents"
)

func handleHTTPEvent(ctx context.Context, event cloudevents.Event, resp *cloudevents.EventResponse) {
	// echo the event back to the source
	resp.RespondWith(200, &event)
}

func main() {
	c, err := kncloudevents.NewDefaultClient()
	if err != nil {
		log.Fatal("Failed to create client, ", err)
	}

	log.Fatal(c.StartReceiver(context.Background(), handleHTTPEvent))
}
