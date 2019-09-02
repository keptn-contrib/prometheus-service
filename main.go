package main

import (
	"context"
	"log"
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
	logger.Debug("Starting handler")
	http.HandleFunc("/", Handler)
	http.ListenAndServe(":8080", nil)

	// listen on port 8081 for CloudEvent
	var env envConfig
	if err := envconfig.Process("", &env); err != nil {
		log.Fatalf("Failed to process env var: %s", err)
	}
	os.Exit(_main(os.Args[1:], env))
}

func _main(args []string, env envConfig) int {

	ctx := context.Background()

	t, err := cloudeventshttp.New(
		cloudeventshttp.WithPort(env.Port),
		cloudeventshttp.WithPath(env.Path),
	)
	if err != nil {
		log.Fatalf("Failed to create transport: %v", err)
	}

	c, err := client.New(t)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	log.Fatalf("Failed to start receiver: %s", c.StartReceiver(ctx, eventhandling.GotEvent))

	return 0
}

// Handler takes the prometheus alert as input
func Handler(rw http.ResponseWriter, req *http.Request) {
	shkeptncontext := ""
	logger := keptnutils.NewLogger(shkeptncontext, "", "prometheus-service")
	logger.Debug("Receiving event from prometheus alertmanager")

	// check event type and start process or forward it to 8081 in case of a Cloud Event
	if false {
		eventhandling.ProcessAndForwardAlertEvent(eventbroker, rw, req, logger, shkeptncontext)
	} else {

		req, err := http.NewRequest("POST", "localhost:8081", req.Body)
		client := &http.Client{}
		resp, err := client.Do(req)
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
