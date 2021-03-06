package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
	"github.com/keptn-contrib/prometheus-service/eventhandling"
	"github.com/keptn-contrib/prometheus-service/utils"
	keptnv2 "github.com/keptn/go-utils/pkg/lib/v0_2_0"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/kelseyhightower/envconfig"
	keptncommon "github.com/keptn/go-utils/pkg/lib/keptn"
)

type envConfig struct {
	// Port on which to listen for cloudevents
	Port int    `envconfig:"RCV_PORT" default:"8081"`
	Path string `envconfig:"RCV_PATH" default:"/"`
}

type ceTest struct {
	Specversion string `json:"specversion" yaml:"specversion"`
}

func main() {
	// listen on port 8080 for any event
	logger := keptncommon.NewLogger("", "", "prometheus-service")
	logger.Debug("Starting server for receiving events on exposed port 8080")
	http.HandleFunc("/", Handler)
	go http.ListenAndServe(":8080", nil)

	// listen on port 8081 for CloudEvent
	var env envConfig
	if err := envconfig.Process("", &env); err != nil {
		logger.Error(fmt.Sprintf("Failed to process env var: %s", err))
	}
	os.Exit(_main(env))
}

func _main(env envConfig) int {
	ctx := context.Background()
	ctx = cloudevents.WithEncodingStructured(ctx)

	p, err := cloudevents.NewHTTP(cloudevents.WithPath(env.Path), cloudevents.WithPort(env.Port))
	if err != nil {
		log.Fatalf("failed to create client, %v", err)
	}
	c, err := cloudevents.NewClient(p)
	if err != nil {
		log.Fatalf("failed to create client, %v", err)
	}
	log.Fatal(c.StartReceiver(ctx, gotEvent))

	return 0
}

func gotEvent(event cloudevents.Event) error {
	var shkeptncontext string
	_ = event.Context.ExtensionAs("shkeptncontext", &shkeptncontext)

	logger := keptncommon.NewLogger(shkeptncontext, event.Context.GetID(), "prometheus-service")

	eventBrokerURL, err := utils.GetEventBrokerURL()
	if err != nil {
		logger.Error(err.Error())
		return err
	}

	keptnHandler, err := keptnv2.NewKeptn(&event, keptncommon.KeptnOpts{
		EventBrokerURL: eventBrokerURL,
	})

	if err != nil {
		return fmt.Errorf("could not create Keptn handler: %v", err)
	}

	if err = eventhandling.NewEventHandler(event, logger, keptnHandler).HandleEvent(); err != nil {
		return err
	}
	return nil

}

// Handler takes request and forwards it to the corresponding event handler
func Handler(rw http.ResponseWriter, req *http.Request) {
	shkeptncontext := uuid.New().String()
	logger := keptncommon.NewLogger(shkeptncontext, "", "prometheus-service")
	logger.Debug("Receiving event which will be dispatched")

	event := ceTest{}
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to read body from requst: %s", err))
		return
	}

	// check event whether event contains specversion to forward it to 8081; otherwise process it as prometheus alert
	if json.Unmarshal(body, &event) != nil || event.Specversion == "" {
		eventhandling.ProcessAndForwardAlertEvent(rw, body, logger, shkeptncontext)
	} else {
		proxyReq, err := http.NewRequest(req.Method, "http://localhost:8081", bytes.NewReader(body))
		proxyReq.Header.Set("Content-Type", "application/cloudevents+json")
		resp, err := http.DefaultClient.Do(proxyReq)
		if err != nil {
			logger.Error("Could not send cloud event: " + err.Error())
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			logger.Error(fmt.Sprintf("Could not process cloud event: Handler returned status %s", resp.Status))
			rw.WriteHeader(500)
		} else {
			logger.Debug("event successfully sent to port 8081")
			rw.WriteHeader(201)
		}
	}
}
