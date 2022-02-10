package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/keptn-contrib/prometheus-service/eventhandling"
	"github.com/keptn-contrib/prometheus-service/utils"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
	"github.com/kelseyhightower/envconfig"
	keptncommon "github.com/keptn/go-utils/pkg/lib/keptn"
	keptnv2 "github.com/keptn/go-utils/pkg/lib/v0_2_0"
)

type ceTest struct {
	Specversion string `json:"specversion" yaml:"specversion"`
}

var (
	env utils.EnvConfig
)

func main() {
	/**
	Note that prometheus-service requires to open multiple ports:
	* 8080 (default port; exposed) - acts as the ingest for prometheus alerts, and also proxies CloudEvents to port 8082
	* 8081 (Keptn distributor) - Port that keptn/distributor is listening too (default port for Keptn distributor)
	* 8082 (CloudEvents; env.Port) - Port that the CloudEvents sdk is listening to; this port is not exposed, but will be used for internal communication
	*/
	logger := keptncommon.NewLogger("", "", utils.ServiceName)

	env = utils.EnvConfig{}

	if err := envconfig.Process("", &env); err != nil {
		logger.Error(fmt.Sprintf("Failed to process env var: %s", err))
	}

	logger.Debug(fmt.Sprintf("Configuration service: %s", env.ConfigurationServiceURL))
	logger.Debug(fmt.Sprintf("Port: %d, Path: %s", env.Port, env.Path))

	// listen on port 8080 for any HTTP request (cloudevents are also handled, but forwarded to env.Port internally)
	logger.Debug("Starting server on port 8080...")
	http.HandleFunc("/", Handler)
	http.HandleFunc("/health", HealthHandler)
	go http.ListenAndServe(":8080", nil)

	// start internal CloudEvents handler (on port env.Port)
	os.Exit(_main(env))
}

func _main(env utils.EnvConfig) int {
	ctx := context.Background()
	ctx = cloudevents.WithEncodingStructured(ctx)

	p, err := cloudevents.NewHTTP(cloudevents.WithPath(env.Path), cloudevents.WithPort(env.Port))
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

	keptnHandler, err := keptnv2.NewKeptn(&event, keptncommon.KeptnOpts{})

	if err != nil {
		return fmt.Errorf("could not create Keptn handler: %v", err)
	}

	logger := keptncommon.NewLogger(shkeptncontext, event.Context.GetID(), utils.ServiceName)

	return eventhandling.NewEventHandler(event, logger, keptnHandler).HandleEvent()
}

// HealthHandler provides a basic health check
func HealthHandler(w http.ResponseWriter, r *http.Request) {
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

// Handler takes all http request and forwards it to the corresponding event handler (e.g., prometheus alert);
// Note: cloudevents are also forwarded
func Handler(rw http.ResponseWriter, req *http.Request) {
	shkeptncontext := uuid.New().String()
	logger := keptncommon.NewLogger(shkeptncontext, "", utils.ServiceName)
	logger.Debug(fmt.Sprintf("%s %s", req.Method, req.URL))
	logger.Debug("Receiving event which will be dispatched")

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to read body from requst: %s", err))
		return
	}

	// try to deserialize the event to check if it contains specversion
	event := ceTest{}
	if err = json.Unmarshal(body, &event); err != nil {
		logger.Debug("Failed to read body: " + err.Error() + "; content=" + string(body))
		return
	}

	// check event whether event contains specversion to forward it to 8081; otherwise process it as prometheus alert
	if event.Specversion == "" {
		// this is a prometheus alert
		eventhandling.ProcessAndForwardAlertEvent(rw, body, logger, shkeptncontext)
	} else {
		// this is a CloudEvent retrieved on port 8080 that needs to be forwarded to 8082 (env.Port)
		forwardPath := fmt.Sprintf("http://localhost:%d%s", env.Port, env.Path)
		logger.Debug("Forwarding cloudevent to " + forwardPath)
		// forward cloudevent to cloudevents lister on env.Port (see main())
		proxyReq, err := http.NewRequest(req.Method, forwardPath, bytes.NewReader(body))
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
