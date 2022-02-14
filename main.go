package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/keptn-contrib/prometheus-service/eventhandling"
	"github.com/keptn-contrib/prometheus-service/utils"
	keptn "github.com/keptn/go-utils/pkg/lib"
	"log"
	"net/http"
	"os"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/kelseyhightower/envconfig"
	keptncommon "github.com/keptn/go-utils/pkg/lib/keptn"
	keptnv2 "github.com/keptn/go-utils/pkg/lib/v0_2_0"
)

var (
	env utils.EnvConfig
)

func main() {
	logger := keptncommon.NewLogger("", "", utils.ServiceName)

	env = utils.EnvConfig{}

	if err := envconfig.Process("", &env); err != nil {
		logger.Error(fmt.Sprintf("Failed to process env var: %s", err))
	}

	logger.Debug(fmt.Sprintf("Configuration service: %s", env.ConfigurationServiceURL))
	logger.Debug(fmt.Sprintf("Port: %d, Path: %s", env.Port, env.Path))

	// start internal CloudEvents handler (on port env.Port)
	os.Exit(_main(env))
}

func _main(env utils.EnvConfig) int {
	ctx := context.Background()
	ctx = cloudevents.WithEncodingStructured(ctx)

	p, err := cloudevents.NewHTTP(cloudevents.WithPath(env.Path), cloudevents.WithPort(env.Port), cloudevents.WithGetHandlerFunc(HTTPGetHandler))
	if err != nil {
		log.Fatalf("failed to create client, %v", err)
	}
	// Create CloudEvents client
	c, err := cloudevents.NewClient(p)
	if err != nil {
		log.Fatalf("failed to create client, %v", err)
	}
	// Start CloudEvents receiver
	log.Fatal(c.StartReceiver(ctx, gotEvent))

	return 0
}

// gotEvent processes an incoming CloudEvent
func gotEvent(event cloudevents.Event) error {
	var shkeptncontext string
	_ = event.Context.ExtensionAs("shkeptncontext", &shkeptncontext)

	logger := keptncommon.NewLogger(shkeptncontext, "", utils.ServiceName)

	// Send prometheus alert
	if event.SpecVersion() == "" {
		eventhandling.ProcessAndForwardAlertEvent(event.Data(), logger, shkeptncontext)
		return nil
	}

	// convert v0.1.4 spec monitoring.configure CloudEvent into a v0.2.0 spec configure-monitoring.triggered CloudEvent
	if event.Type() == keptn.ConfigureMonitoringEventType {
		event.SetType(keptnv2.GetTriggeredEventType(keptnv2.ConfigureMonitoringTaskName))
	}

	keptnHandler, err := keptnv2.NewKeptn(&event, keptncommon.KeptnOpts{})

	if err != nil {
		return fmt.Errorf("could not create Keptn handler: %v", err)
	}

	return eventhandling.NewEventHandler(event, logger, keptnHandler).HandleEvent()
}

// HTTPGetHandler will handle all requests for '/health' and '/ready'
func HTTPGetHandler(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/health":
		healthEndpointHandler(w, r)
	case "/ready":
		healthEndpointHandler(w, r)
	default:
		endpointNotFoundHandler(w, r)
	}
}

// HealthHandler rerts a basic health check back
func healthEndpointHandler(w http.ResponseWriter, r *http.Request) {
	type StatusBody struct {
		Status string `json:"status"`
	}

	status := StatusBody{Status: "OK"}

	body, _ := json.Marshal(status)

	w.Header().Set("content-type", "application/json")

	_, err := w.Write(body)
	if err != nil {
		log.Println(err)
	}
}

// endpointNotFoundHandler will return 404 for requests
func endpointNotFoundHandler(w http.ResponseWriter, r *http.Request) {
	type StatusBody struct {
		Status string `json:"status"`
	}

	status := StatusBody{Status: "NOT FOUND"}

	body, _ := json.Marshal(status)

	w.Header().Set("content-type", "application/json")

	_, err := w.Write(body)
	if err != nil {
		log.Println(err)
	}
}
