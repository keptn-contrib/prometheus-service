package eventhandling

import (
	"context"
	"errors"
	"fmt"
	"github.com/keptn-contrib/prometheus-service/utils/prometheus"
	"gopkg.in/yaml.v2"
	"log"
	"net/url"
	"strings"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/keptn-contrib/prometheus-service/utils"

	keptnv2 "github.com/keptn/go-utils/pkg/lib/v0_2_0"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// GetSliEventHandler is responsible for processing configure monitoring events
type GetSliEventHandler struct {
	event        cloudevents.Event
	keptnHandler *keptnv2.Keptn
	kubeClient   *kubernetes.Clientset
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

	// send started event
	_, err = eh.keptnHandler.SendTaskStartedEvent(eventData, utils.ServiceName)
	if err != nil {
		errMsg := fmt.Errorf("failed to send task started CloudEvent: %w", err)
		log.Println(errMsg.Error())
		return err
	}

	// helper function to log an error and send an appropriate finished event
	sendFinishedErrorEvent := func(err error) error {
		log.Printf("sending errored finished event: %s", err.Error())

		_, sendError := eh.keptnHandler.SendTaskFinishedEvent(&keptnv2.EventData{
			Status:  keptnv2.StatusErrored,
			Result:  keptnv2.ResultFailed,
			Message: err.Error(),
		}, utils.ServiceName)

		// TODO: Maybe log error to console

		return sendError
	}

	// get prometheus API URL for the provided Project from Kubernetes Config Map
	prometheusAPIURL, err := getPrometheusAPIURL(eventData.Project, eh.kubeClient.CoreV1())
	if err != nil {
		return sendFinishedErrorEvent(fmt.Errorf("unable to get prometheus api URL: %w", err))
	}

	// determine deployment type based on what lighthouse-service is providing
	deployment := eventData.Deployment // "canary", "primary" or "" (or "direct" or "user_managed")
	// fallback: get deployment type from labels
	if deploymentLabel, ok := eventData.Labels["deployment"]; deployment == "" && !ok {
		log.Println("Warning: no deployment type specified in event, defaulting to \"primary\"")
		deployment = "primary"
	} else if ok {
		log.Println("Deployment was not set, but label exist. Using label from event")
		deployment = deploymentLabel
	}

	// create a new Prometheus Handler
	prometheusHandler := prometheus.NewPrometheusHandler(
		prometheusAPIURL,
		&eventData.EventData,
		deployment,
		eventData.Labels,
		eventData.GetSLI.CustomFilters,
	)

	// get SLI queries (from SLI.yaml)
	projectCustomQueries, err := getCustomQueries(eh.keptnHandler, eventData.Project, eventData.Stage, eventData.Service)
	if err != nil {
		return sendFinishedErrorEvent(
			fmt.Errorf("unable to retrieve custom queries for project %s: %w", eventData.Project, err),
		)
	}

	// only apply queries if they contain anything
	if projectCustomQueries != nil {
		prometheusHandler.CustomQueries = projectCustomQueries
	}

	// retrieve metrics from prometheus
	sliResults := retrieveMetrics(prometheusHandler, eventData)

	// If we hand any problem retrieving an SLI value, we set the result of the overall .finished event
	// to Warning, if all fail ResultFailed is set for the event
	finalSLIEventResult := keptnv2.ResultPass

	if len(sliResults) > 0 {
		sliResultsFailed := 0
		for _, sliResult := range sliResults {
			if !sliResult.Success {
				sliResultsFailed++
			}
		}

		if sliResultsFailed > 0 && sliResultsFailed < len(sliResults) {
			finalSLIEventResult = keptnv2.ResultWarning
		} else if sliResultsFailed == len(sliResults) {
			finalSLIEventResult = keptnv2.ResultFailed
		}
	}

	// construct finished event data
	getSliFinishedEventData := &keptnv2.GetSLIFinishedEventData{
		EventData: keptnv2.EventData{
			Status: keptnv2.StatusSucceeded,
			Result: finalSLIEventResult,
		},
		GetSLI: keptnv2.GetSLIFinished{
			IndicatorValues: sliResults,
			Start:           eventData.GetSLI.Start,
			End:             eventData.GetSLI.End,
		},
	}

	if getSliFinishedEventData.EventData.Result == keptnv2.ResultFailed {
		getSliFinishedEventData.EventData.Message = "unable to retrieve metrics"
	}

	// send get-sli.finished event with SLI DATA
	_, err = eh.keptnHandler.SendTaskFinishedEvent(getSliFinishedEventData, utils.ServiceName)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to send task finished CloudEvent (%s), aborting...", err.Error())
		log.Println(errMsg)
		return err
	}

	return nil
}

func retrieveMetrics(prometheusHandler *prometheus.Handler, eventData *keptnv2.GetSLITriggeredEventData) []*keptnv2.SLIResult {
	log.Printf("Retrieving Prometheus metrics")

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
		} else {
			sliResults = append(sliResults, &keptnv2.SLIResult{
				Metric:  indicator,
				Value:   sliValue,
				Success: true,
			})
		}
	}

	return sliResults
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

	secretName := fmt.Sprintf("prometheus-credentials-%s", project)

	secret, err := kubeClient.Secrets(env.PodNamespace).Get(context.TODO(), secretName, metav1.GetOptions{})

	// fallback: return cluster-internal prometheus URL (configured via PrometheusEndpoint environment variable)
	// in case no secret has been created for this project
	if err != nil {
		log.Println("Could not retrieve or read secret (" + err.Error() + ") for project " + project + ". Using default: " + env.PrometheusEndpoint)
		return env.PrometheusEndpoint, nil
	}

	pc := prometheusCredentials{}

	// Read Prometheus config from Kubernetes secret as strings
	// Example: keptn create secret prometheus-credentials-<project> --scope="keptn-prometheus-service" --from-literal="PROMETHEUS_USER=$PROMETHEUS_USER" --from-literal="PROMETHEUS_PASSWORD=$PROMETHEUS_PASSWORD" --from-literal="PROMETHEUS_URL=$PROMETHEUS_URL"
	prometheusURL, errURL := utils.ReadK8sSecretAsString(env.PodNamespace, secretName, "PROMETHEUS_URL")
	prometheusUser, errUser := utils.ReadK8sSecretAsString(env.PodNamespace, secretName, "PROMETHEUS_USER")
	prometheusPassword, errPassword := utils.ReadK8sSecretAsString(env.PodNamespace, secretName, "PROMETHEUS_PASSWORD")

	if errURL == nil && errUser == nil && errPassword == nil {
		// found! using it
		pc.URL = prometheusURL
		pc.User = prometheusUser
		pc.Password = prometheusPassword
	} else {
		// deprecated: try to use legacy approach
		err = yaml.Unmarshal(secret.Data["prometheus-credentials"], &pc)

		if err != nil {
			log.Println("Could not parse credentials for external prometheus instance: " + err.Error())
			return "", errors.New("invalid credentials format found in secret 'prometheus-credentials-" + project)
		}

		// warn the user to migrate their credentials
		log.Printf("Warning: Please migrate your prometheus credentials for project %s. ", project)
		log.Printf("See https://github.com/keptn-contrib/prometheus-service/issues/274 for more information.\n")
	}

	log.Println("Using external prometheus instance for project " + project + ": " + pc.URL)
	return generatePrometheusURL(&pc), nil
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
