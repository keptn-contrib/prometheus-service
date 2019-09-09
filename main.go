package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/keptn-contrib/prometheus-service/eventhandling"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/cloudevents/sdk-go/pkg/cloudevents/client"
	cloudeventshttp "github.com/cloudevents/sdk-go/pkg/cloudevents/transport/http"
	"github.com/kelseyhightower/envconfig"
	keptnutils "github.com/keptn/go-utils/pkg/utils"
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
	shkeptncontext := ""
	logger := keptnutils.NewLogger(shkeptncontext, "", "prometheus-service")
	logger.Debug("Starting server for receiving events on exposed port 8080")
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
	logger.Debug("Starting server for receiving Cloud Events on 8081 for internal use")
	logger.Error(fmt.Sprintf("Failed to start receiver: %s", c.StartReceiver(ctx, eventhandling.GotEvent)))

	return 0
}

// Handler takes request and forwards it to the corresponding event handler
func Handler(rw http.ResponseWriter, req *http.Request) {
	shkeptncontext := ""
	logger := keptnutils.NewLogger(shkeptncontext, "", "prometheus-service")
	logger.Debug("Receiving event which will be dispatched")

	event := ceTest{}
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to read body from requst: %s", err))
		return
	}

	// check event whether event contains specversion to forward it to 8081; otherwise process it as prometheus alert
	if json.Unmarshal(body, &event) != nil || event.Specversion == "" {
		eventhandling.ProcessAndForwardAlertEvent(rw, req, logger, shkeptncontext)
	} else {
		proxyReq, err := http.NewRequest(req.Method, "http://localhost:8081", bytes.NewReader(body))
		proxyReq.Header.Set("Content-Type", "application/cloudevents+json")
		resp, err := http.DefaultClient.Do(proxyReq)
		if err != nil {
			logger.Error("Could not send cloud event: " + err.Error())
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != 202 {
			logger.Error(fmt.Sprintf("Could not send cloud event: %s", err.Error()))
			rw.WriteHeader(500)
		} else {
			logger.Debug("Event successfully sent to port 8081")
			rw.WriteHeader(201)
		}
	}
}
