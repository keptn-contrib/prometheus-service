package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/cloudevents/sdk-go/pkg/cloudevents/client"
	cloudeventshttp "github.com/cloudevents/sdk-go/pkg/cloudevents/transport/http"
	"github.com/kelseyhightower/envconfig"
	"github.com/keptn-contrib/prometheus-service/eventhandling"
	keptnutils "github.com/keptn/go-utils/pkg/utils"
)

type envConfig struct {
	// Port on which to listen for cloudevents
	Port int    `envconfig:"RCV_PORT" default:"8081"`
	Path string `envconfig:"RCV_PATH" default:"/"`
}

const eventbroker = "EVENTBROKER"

func main() {
	// listen on port 8080 for any event
	shkeptncontext := ""
	logger := keptnutils.NewLogger(shkeptncontext, "", "prometheus-service")
	logger.Debug("Starting server for receiving events on 8080")
	http.HandleFunc("/", Handler)
	go http.ListenAndServe(":8080", nil)

	// listen on port 8081 for CloudEvent
	var env envConfig
	if err := envconfig.Process("", &env); err != nil {
		logger.Error(fmt.Sprintf("Failed to process env var: %s", err))
	}
	os.Exit(_main(os.Args[1:], env))
}

func _main(args []string, env envConfig) int {
	shkeptncontext := ""
	logger := keptnutils.NewLogger(shkeptncontext, "", "prometheus-service")

	ctx := context.Background()

	t, err := cloudeventshttp.New(
		cloudeventshttp.WithPort(env.Port),
		cloudeventshttp.WithPath(env.Path),
	)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to create transport: %v", err))
	}

	c, err := client.New(t)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to create client: %v", err))
	}
	logger.Debug("Starting server for receiving Cloud Events on 8081")
	logger.Error(fmt.Sprintf("Failed to start receiver: %s", c.StartReceiver(ctx, eventhandling.GotEvent)))

	return 0
}

// Handler takes request and forwards it to the corresponding event handler
func Handler(rw http.ResponseWriter, req *http.Request) {
	shkeptncontext := ""
	logger := keptnutils.NewLogger(shkeptncontext, "", "prometheus-service")
	logger.Debug("Receiving event from prometheus alertmanager")

	// check event type and start process or forward it to 8081 in case of a Cloud Event
	if false {
		eventhandling.ProcessAndForwardAlertEvent(eventbroker, rw, req, logger, shkeptncontext)
	} else {

		req, err := http.NewRequest("POST", "http://localhost:8081", req.Body)
		req.Header.Set("Content-Type", "application/cloudevents+json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			logger.Error("Could not send cloud event: " + err.Error())
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			logger.Error("Could not send cloud event: " + err.Error())
			rw.WriteHeader(500)
		} else {
			logger.Debug("Event successfully sent to port 8081")
			rw.WriteHeader(201)
		}
	}
}
