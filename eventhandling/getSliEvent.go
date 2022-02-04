package eventhandling

import (
	"errors"
	"fmt"
	"gopkg.in/yaml.v2"
	"log"
	"math"
	"net/url"
	"strings"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/types"
	"github.com/keptn-contrib/prometheus-service/utils"
	keptncommon "github.com/keptn/go-utils/pkg/lib/keptn"
	keptnv2 "github.com/keptn/go-utils/pkg/lib/v0_2_0"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
)

// GetSliEventHandler is responsible for processing configure monitoring events
type GetSliEventHandler struct {
	event        cloudevents.Event
	keptnHandler *keptnv2.Keptn
}

type prometheusCredentials struct {
	URL      string `json:"url" yaml:"url"`
	User     string `json:"user" yaml:"user"`
	Password string `json:"password" yaml:"password"`
}

// HandleEvent processes an event
func (eh GetSliEventHandler) HandleEvent() error {
	eventData := &keptnv2.GetSLITriggeredEventData{}
	err := eh.event.DataAs(eventData)
	if err != nil {
		return err
	}

	// don't continue if SLIProvider is not prometheus
	if eventData.GetSLI.SLIProvider != "prometheus" {
		return nil
	}

	// get shkeptncontext
	keptnCtx, err := types.ToString(eh.event.Context.GetExtensions()["shkeptncontext"])
	if err != nil {
		return fmt.Errorf("could not determine keptnContext of input event: %s", err.Error())
	}

	// create empty SLI Results Array
	var sliResults = []*keptnv2.SLIResult{}

	// 1: send .started event, indicating that we accepted it
	if err = sendGetSLIStartedEvent(eh.event, eventData, keptnCtx); err != nil {
		return sendGetSLIFinishedEvent(eh.event, eventData, sliResults, err, keptnCtx)
	}

	// 2: try to fetch metrics into sliResults
	if sliResults, err = retrieveMetrics(eh.event, eventData); err != nil {
		// failed to fetch metrics, send a finished event with the error
		return sendGetSLIFinishedEvent(eh.event, eventData, sliResults, err, keptnCtx)
	}

	// 3: success; send .finished event with metrics (sliResults)
	return sendGetSLIFinishedEvent(eh.event, eventData, sliResults, nil, keptnCtx)
}

func retrieveMetrics(event cloudevents.Event, eventData *keptnv2.GetSLITriggeredEventData) ([]*keptnv2.SLIResult, error) {
	log.Printf("Retrieving Prometheus metrics")

	clusterConfig, err := rest.InClusterConfig()
	if err != nil {
		log.Println("could not create Kubernetes cluster config")
		return nil, errors.New("could not create Kubernetes client")
	}

	kubeClient, err := kubernetes.NewForConfig(clusterConfig)
	if err != nil {
		log.Println("could not create Kubernetes client")
		return nil, errors.New("could not create Kubernetes client")
	}

	prometheusAPIURL, err := getPrometheusAPIURL(eventData.Project, kubeClient.CoreV1())
	if err != nil {
		return nil, err
	}

	keptnHandler, err := keptnv2.NewKeptn(&event, keptncommon.KeptnOpts{})
	if err != nil {
		return nil, err
	}

	// Create a new Prometheus Handler
	prometheusHandler := utils.NewPrometheusHandler(
		prometheusAPIURL,
		&eventData.EventData,
		eventData.GetSLI.CustomFilters,
	)

	projectCustomQueries, err := getCustomQueries(keptnHandler, eventData.Project, eventData.Stage, eventData.Service)
	if err != nil {
		log.Println("retrieveMetrics: Failed to get custom queries for project " + eventData.Project)
		log.Println(err.Error())
		return nil, err
	}

	if projectCustomQueries != nil {
		prometheusHandler.CustomQueries = projectCustomQueries
	}

	var sliResults []*keptnv2.SLIResult

	for _, indicator := range eventData.GetSLI.Indicators {
		log.Println("retrieveMetrics: Fetching indicator: " + indicator)
		sliValue, err := prometheusHandler.GetSLIValue(indicator, eventData.GetSLI.Start, eventData.GetSLI.End)
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

func getCustomQueries(keptnHandler *keptnv2.Keptn, project string, stage string, service string) (map[string]string, error) {
	log.Println("Checking for custom SLI queries")

	customQueries, err := keptnHandler.GetSLIConfiguration(project, stage, service, utils.SliResourceURI)
	if err != nil {
		return nil, err
	}

	return customQueries, nil
}

// getPrometheusAPIURL fetches the prometheus API URL for the provided project (e.g., from Kubernetes configmap)
func getPrometheusAPIURL(project string, kubeClient v1.CoreV1Interface) (string, error) {
	log.Println("Checking if external prometheus instance has been defined for project " + project)
	secret, err := kubeClient.Secrets(env.PodNamespace).Get("prometheus-credentials-"+project, metav1.GetOptions{})

	// return cluster-internal prometheus URL if no secret has been found
	if err != nil {
		log.Println("could not retrieve or read secret: " + err.Error())
		log.Println("No external prometheus instance defined for project " + project + ". Using default: " + env.PrometheusEndpoint)
		return env.PrometheusEndpoint, nil
	}

	pc := &prometheusCredentials{}
	err = yaml.Unmarshal(secret.Data["prometheus-credentials"], pc)

	if err != nil {
		log.Println("Could not parse credentials for external prometheus instance: " + err.Error())
		return "", errors.New("invalid credentials format found in secret 'prometheus-credentials-" + project)
	}
	log.Println("Using external prometheus instance for project " + project + ": " + pc.URL)
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

	source, _ := url.Parse(utils.ServiceName)

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
	source, _ := url.Parse(utils.ServiceName)
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
