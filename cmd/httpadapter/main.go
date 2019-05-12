package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	cloudevents "github.com/cloudevents/sdk-go"
	"github.com/cloudevents/sdk-go/pkg/cloudevents/client"
	"github.com/cloudevents/sdk-go/pkg/cloudevents/types"
	"github.com/knative/eventing-sources/pkg/kncloudevents"
)

const (
	envPort    = "PORT"
	envSinkURI = "SINK_URI"
)

type Adapter struct {
	client client.Client
}

func (a *Adapter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	b, err := ioutil.ReadAll(req.Body)
	if err != nil {
		w.WriteHeader(400)
		fmt.Fprintf(w, err.Error())
		return
	}

	event := cloudevents.Event{
		Context: cloudevents.EventContextV02{
			ID:     "1234",
			Type:   "dev.knative.source.http",
			Source: *types.ParseURLRef("http://owenou.com"),
		}.AsV02(),
		Data: map[string]string{"body": string(b)},
	}

	resp, err := a.client.Send(context.TODO(), event)
	if err != nil {
		w.WriteHeader(400)
		fmt.Fprintf(w, err.Error())
		return
	}

	if resp != nil {
		fmt.Fprintf(w, resp.String())
	}
}

func main() {
	sinkURI := os.Getenv(envSinkURI)
	if sinkURI == "" {
		log.Fatalf("No sink given")
	}

	port := os.Getenv(envPort)
	if port == "" {
		port = "8080"
	}

	client, err := kncloudevents.NewDefaultClient(sinkURI)
	if err != nil {
		log.Fatalf("No sink given")
	}

	http.Handle("/", &Adapter{client: client})
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
