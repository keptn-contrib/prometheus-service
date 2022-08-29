package main

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/keptn-contrib/prometheus-service/eventhandling"
	"github.com/keptn-contrib/prometheus-service/utils"
	"github.com/keptn/go-utils/pkg/sdk"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"log"
	"net/http"
	"os"

	keptnevents "github.com/keptn/go-utils/pkg/lib"
	keptncommon "github.com/keptn/go-utils/pkg/lib/keptn"
	keptnv2 "github.com/keptn/go-utils/pkg/lib/v0_2_0"
)

var (
	env utils.EnvConfig
)

const serviceName = "prometheus-service"
const envVarLogLevel = "LOG_LEVEL"
const monitoringTriggeredEvent = keptnevents.ConfigureMonitoringEventType
const getSliTriggeredEvent = "sh.keptn.event.get-sli.triggered"

func main() {
	if os.Getenv(envVarLogLevel) != "" {
		logLevel, err := logrus.ParseLevel(os.Getenv(envVarLogLevel))
		if err != nil {
			logrus.WithError(err).Error("could not parse log level provided by 'LOG_LEVEL' env var")
			logrus.SetLevel(logrus.InfoLevel)
		} else {
			logrus.SetLevel(logLevel)
		}
	}

	log.Printf("Starting %s", serviceName)

	// Creating an HTTP listener on port 8080 to receive alerts from Prometheus directly
	http.HandleFunc("/", HTTPGetHandler)
	go func() {
		log.Println("Starting alert manager endpoint")
		err := http.ListenAndServe(":8080", nil)
		if err != nil {
			log.Fatalf("Error with HTTP server: %e", err)
		}
	}()

	clusterConfig, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("unable to create kubernetes cluster config: %e", err)
	}

	kubeClient, err := kubernetes.NewForConfig(clusterConfig)
	if err != nil {
		log.Fatalf("unable to create kubernetes client: %e", err)
	}

	log.Fatal(sdk.NewKeptn(
		serviceName,
		sdk.WithTaskHandler(
			monitoringTriggeredEvent,
			eventhandling.NewConfigureMonitoringEventHandler(),
			prometheusTypeFilter),
		sdk.WithTaskHandler(
			getSliTriggeredEvent,
			eventhandling.NewGetSliEventHandler(*kubeClient),
			prometheusSLIProviderFilter),
		sdk.WithLogger(logrus.New()),
	).Start())
}

// prometheusSLIProviderFilter filters get-sli.triggered events for Prometheus
func prometheusSLIProviderFilter(keptnHandle sdk.IKeptn, event sdk.KeptnEvent) bool {
	data := &keptnv2.GetSLITriggeredEventData{}
	if err := keptnv2.Decode(event.Data, data); err != nil {
		keptnHandle.Logger().Errorf("Could not parse get-sli.triggered event: %s", err.Error())
		return false
	}

	return data.GetSLI.SLIProvider == "prometheus"
}

// prometheusTypeFilter filters monitoring.configure events for Prometheus
func prometheusTypeFilter(keptnHandle sdk.IKeptn, event sdk.KeptnEvent) bool {
	data := &keptnevents.ConfigureMonitoringEventData{}
	if err := keptnv2.Decode(event.Data, data); err != nil {
		keptnHandle.Logger().Errorf("Could not parse monitoring.configure event: %s", err.Error())
		return false
	}

	return data.Type == "prometheus"
}

// HTTPGetHandler will handle all requests for '/health' and '/ready'
func HTTPGetHandler(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/":
		shkeptncontext := uuid.New().String()
		logger := keptncommon.NewLogger(shkeptncontext, "", utils.ServiceName)

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to read body from requst: %s", err))
			return
		}

		eventhandling.ProcessAndForwardAlertEvent(w, body, logger, shkeptncontext)
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
