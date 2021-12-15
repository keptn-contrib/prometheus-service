package main

import (
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"strings"

	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/cloudevents/sdk-go/v2/types"
	"github.com/google/uuid"
	"github.com/kelseyhightower/envconfig"
	"github.com/keptn-contrib/prometheus-service/eventhandling"
	"github.com/keptn-contrib/prometheus-service/utils"
	"gopkg.in/yaml.v2"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	keptncommon "github.com/keptn/go-utils/pkg/lib/keptn"
	keptnv2 "github.com/keptn/go-utils/pkg/lib/v0_2_0"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

const sliResourceURI = "prometheus/sli.yaml"
const serviceName = "prometheus-service"

type envConfig struct {
	// Port on which to listen for cloudevents
	Port                    int    `envconfig:"RCV_PORT" default:"8082"` // Note: must not be 8080 and not 8081
	Path                    string `envconfig:"RCV_PATH" default:"/"`
	ConfigurationServiceURL string `envconfig:"CONFIGURATION_SERVICE" default:""`
}

type prometheusCredentials struct {
	URL      string `json:"url" yaml:"url"`
	User     string `json:"user" yaml:"user"`
	Password string `json:"password" yaml:"password"`
}

type ceTest struct {
	Specversion string `json:"specversion" yaml:"specversion"`
}

var (
	namespace          = os.Getenv("POD_NAMESPACE")
	prometheusEndpoint = os.Getenv("PROMETHEUS_ENDPOINT")
	env                envConfig
)

func main() {
	/**
	Note that prometheus-service requires to open multiple ports:
	* 8080 (default port; exposed) - acts as the ingest for prometheus alerts, and also proxies CloudEvents to port 8082
	* 8081 (Keptn distributor) - Port that keptn/distributor is listening too (default port for Keptn distributor)
	* 8082 (CloudEvents; env.Port) - Port that the CloudEvents sdk is listening to; this port is not exposed, but will be used for internal communication
	*/
	logger := keptncommon.NewLogger("", "", "prometheus-service")

	env = envConfig{}

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

	if event.Type() == keptnv2.GetTriggeredEventType(keptnv2.GetSLITaskName) {
		return processEvent(event)
	}

	logger := keptncommon.NewLogger(shkeptncontext, event.Context.GetID(), "prometheus-service")

	keptnHandler, err := keptnv2.NewKeptn(&event, keptncommon.KeptnOpts{})

	if err != nil {
		return fmt.Errorf("could not create Keptn handler: %v", err)
	}

	if err = eventhandling.NewEventHandler(event, logger, keptnHandler).HandleEvent(); err != nil {
		return err
	}
	return nil

}

// HealthHandler rerts a basic health check back
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

// Handler takes request and forwards it to the corresponding event handler; Note: cloudevents are also forwarded
func Handler(rw http.ResponseWriter, req *http.Request) {
	shkeptncontext := uuid.New().String()
	logger := keptncommon.NewLogger(shkeptncontext, "", "prometheus-service")
	logger.Debug(fmt.Sprintf("%s %s", req.Method, req.URL))
	logger.Debug("Receiving event which will be dispatched")

	event := ceTest{}
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to read body from requst: %s", err))
		return
	}

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

func processEvent(event cloudevents.Event) error {

	eventData := &keptnv2.GetSLITriggeredEventData{}
	err := event.DataAs(eventData)
	if err != nil {
		return err
	}

	keptnCtx, err := types.ToString(event.Context.GetExtensions()["shkeptncontext"])
	if err != nil {
		return fmt.Errorf("could not determine keptnContext of input event: %s", err.Error())
	}

	log := keptncommon.NewLogger(keptnCtx, event.Context.GetID(), serviceName)

	// don't continue if SLIProvider is not prometheus
	if eventData.GetSLI.SLIProvider != "prometheus" {
		return nil
	}

	// 1: send .started event
	var sliResults = []*keptnv2.SLIResult{}
	if err = sendGetSLIStartedEvent(event, eventData, keptnCtx); err != nil {
		return sendGetSLIFinishedEvent(event, eventData, sliResults, err, keptnCtx)
	}

	// 2: try to fetch metrics
	if sliResults, err = retrieveMetrics(event, eventData, log); err != nil {
		return sendGetSLIFinishedEvent(event, eventData, sliResults, err, keptnCtx)
	}

	// 3: send .finished event
	return sendGetSLIFinishedEvent(event, eventData, sliResults, nil, keptnCtx)
}

func retrieveMetrics(event cloudevents.Event, eventData *keptnv2.GetSLITriggeredEventData, log keptncommon.LoggerInterface) ([]*keptnv2.SLIResult, error) {
	log.Info("Retrieving Prometheus metrics")

	clusterConfig, err := rest.InClusterConfig()
	if err != nil {
		log.Error("could not create Kubernetes cluster config")
		return nil, errors.New("could not create Kubernetes client")
	}

	kubeClient, err := kubernetes.NewForConfig(clusterConfig)
	if err != nil {
		log.Error("could not create Kubernetes client")
		return nil, errors.New("could not create Kubernetes client")
	}

	prometheusApiURL, err := getPrometheusApiURL(eventData.Project, kubeClient.CoreV1(), log)
	if err != nil {
		return nil, err
	}

	keptnHandler, err := keptnv2.NewKeptn(&event, keptncommon.KeptnOpts{})
	if err != nil {
		return nil, err
	}

	prometheusHandler := utils.NewPrometheusHandler(prometheusApiURL, eventData.Project, eventData.Stage, eventData.Service, eventData.GetSLI.CustomFilters)

	projectCustomQueries, err := getCustomQueries(keptnHandler, eventData.Project, eventData.Stage, eventData.Service, log)
	if err != nil {
		log.Error("Failed to get custom queries for project " + eventData.Project)
		log.Error(err.Error())
		return nil, err
	}

	if projectCustomQueries != nil {
		prometheusHandler.CustomQueries = projectCustomQueries
	}

	var sliResults []*keptnv2.SLIResult

	for _, indicator := range eventData.GetSLI.Indicators {
		log.Info("Fetching indicator: " + indicator)
		sliValue, err := prometheusHandler.GetSLIValue(indicator, eventData.GetSLI.Start, eventData.GetSLI.End, log)
		if err != nil {
			sliResults = append(sliResults, &keptnv2.SLIResult{
				Metric:  indicator,
				Value:   0,
				Success: false,
				Message: err.Error(),
			})
		} else if math.IsNaN(sliValue) {
			sliResults = append(sliResults, &keptnv2.SLIResult{
				Metric:  indicator,
				Value:   0,
				Success: false,
				Message: "SLI value is NaN",
			})
		} else {
			sliResults = append(sliResults, &keptnv2.SLIResult{
				Metric:  indicator,
				Value:   sliValue,
				Success: true,
			})
		}
	}
	return sliResults, nil
}

func getCustomQueries(keptnHandler *keptnv2.Keptn, project string, stage string, service string, logger keptncommon.LoggerInterface) (map[string]string, error) {
	logger.Info("Checking for custom SLI queries")

	customQueries, err := keptnHandler.GetSLIConfiguration(project, stage, service, sliResourceURI)
	if err != nil {
		return nil, err
	}

	return customQueries, nil
}

func getPrometheusApiURL(project string, kubeClient v1.CoreV1Interface, logger keptncommon.LoggerInterface) (string, error) {
	logger.Info("Checking if external prometheus instance has been defined for project " + project)
	secret, err := kubeClient.Secrets(namespace).Get("prometheus-credentials-"+project, metav1.GetOptions{})

	// return cluster-internal prometheus URL if no secret has been found
	if err != nil {
		logger.Info("could not retrieve or read secret: " + err.Error())
		logger.Info("No external prometheus instance defined for project " + project + ". Using default: " + prometheusEndpoint)
		return prometheusEndpoint, nil
	}

	pc := &prometheusCredentials{}
	err = yaml.Unmarshal(secret.Data["prometheus-credentials"], pc)

	if err != nil {
		logger.Error("Could not parse credentials for external prometheus instance: " + err.Error())
		return "", errors.New("invalid credentials format found in secret 'prometheus-credentials-" + project)
	}
	logger.Info("Using external prometheus instance for project " + project + ": " + pc.URL)
	prometheusURL := generatePrometheusURL(pc)

	return prometheusURL, nil
}

func generatePrometheusURL(pc *prometheusCredentials) string {
	prometheusURL := pc.URL

	credentialsString := ""

	if pc.User != "" && pc.Password != "" {
		credentialsString = url.QueryEscape(pc.User) + ":" + url.QueryEscape(pc.Password) + "@"
	}
	if strings.HasPrefix(prometheusURL, "https://") {
		prometheusURL = strings.TrimPrefix(prometheusURL, "https://")
		prometheusURL = "https://" + credentialsString + prometheusURL
	} else if strings.HasPrefix(prometheusURL, "http://") {
		prometheusURL = strings.TrimPrefix(prometheusURL, "http://")
		prometheusURL = "http://" + credentialsString + prometheusURL
	} else {
		// assume https transport
		prometheusURL = "https://" + credentialsString + prometheusURL
	}
	return strings.Replace(prometheusURL, " ", "", -1)
}

func sendGetSLIStartedEvent(inputEvent cloudevents.Event, eventData *keptnv2.GetSLITriggeredEventData, keptnContext interface{}) error {

	source, _ := url.Parse(serviceName)

	getSLIStartedEvent := keptnv2.GetSLIStartedEventData{
		EventData: keptnv2.EventData{
			Project: eventData.Project,
			Stage:   eventData.Stage,
			Service: eventData.Service,
			Labels:  eventData.Labels,
			Status:  keptnv2.StatusSucceeded,
			Result:  keptnv2.ResultPass,
		},
	}

	event := cloudevents.NewEvent()
	event.SetType(keptnv2.GetStartedEventType(keptnv2.GetSLITaskName))
	event.SetSource(source.String())
	event.SetDataContentType(cloudevents.ApplicationJSON)
	event.SetExtension("shkeptncontext", keptnContext)
	event.SetExtension("triggeredid", inputEvent.ID())
	event.SetData(cloudevents.ApplicationJSON, getSLIStartedEvent)

	return sendEvent(event)
}

func sendGetSLIFinishedEvent(inputEvent cloudevents.Event, eventData *keptnv2.GetSLITriggeredEventData, indicatorValues []*keptnv2.SLIResult, err error, keptnContext interface{}) error {
	source, _ := url.Parse(serviceName)
	var status = keptnv2.StatusSucceeded
	var result = keptnv2.ResultPass
	var message = ""

	if err != nil {
		status = keptnv2.StatusErrored
		result = keptnv2.ResultFailed
		message = err.Error()
	}

	getSLIEvent := keptnv2.GetSLIFinishedEventData{
		EventData: keptnv2.EventData{
			Project: eventData.Project,
			Stage:   eventData.Stage,
			Service: eventData.Service,
			Labels:  eventData.Labels,
			Status:  status,
			Result:  result,
			Message: message,
		},
		GetSLI: keptnv2.GetSLIFinished{
			IndicatorValues: indicatorValues,
			Start:           eventData.GetSLI.Start,
			End:             eventData.GetSLI.End,
		},
	}

	event := cloudevents.NewEvent()
	event.SetType(keptnv2.GetFinishedEventType(keptnv2.GetSLITaskName))
	event.SetSource(source.String())
	event.SetDataContentType(cloudevents.ApplicationJSON)
	event.SetExtension("shkeptncontext", keptnContext)
	event.SetExtension("triggeredid", inputEvent.ID())
	event.SetData(cloudevents.ApplicationJSON, getSLIEvent)

	return sendEvent(event)
}

func sendEvent(event cloudevents.Event) error {
	keptnHandler, err := keptnv2.NewKeptn(&event, keptncommon.KeptnOpts{})
	if err != nil {
		return err
	}

	return keptnHandler.SendCloudEvent(event)
}
