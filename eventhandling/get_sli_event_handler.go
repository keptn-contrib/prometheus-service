package eventhandling

import (
	"context"
	"errors"
	"fmt"
	"github.com/kelseyhightower/envconfig"
	"github.com/keptn-contrib/prometheus-service/utils/prometheus"
	"github.com/keptn/go-utils/pkg/api/models"
	api "github.com/keptn/go-utils/pkg/api/utils"
	keptncommon "github.com/keptn/go-utils/pkg/lib/keptn"
	"github.com/keptn/go-utils/pkg/sdk"
	"gopkg.in/yaml.v2"
	"k8s.io/client-go/kubernetes"
	"log"
	"net/url"
	"strings"

	"github.com/keptn-contrib/prometheus-service/utils"

	keptnv2 "github.com/keptn/go-utils/pkg/lib/v0_2_0"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// GetSliEventHandler is responsible for processing configure monitoring events
type GetSliEventHandler struct {
	kubeClient kubernetes.Clientset
}

// NewGetSliEventHandler creates a new TriggeredEventHandler
func NewGetSliEventHandler(kubeClient kubernetes.Clientset) *GetSliEventHandler {
	return &GetSliEventHandler{
		kubeClient: kubeClient,
	}
}

type prometheusCredentials struct {
	URL      string `json:"url" yaml:"url"`
	User     string `json:"user" yaml:"user"`
	Password string `json:"password" yaml:"password"`
}

var env utils.EnvConfig

// Execute processes an event
func (eh GetSliEventHandler) Execute(k sdk.IKeptn, event sdk.KeptnEvent) (interface{}, *sdk.Error) {
	if err := envconfig.Process("", &env); err != nil {
		k.Logger().Error("Failed to process env var: " + err.Error())
	}

	eventData := &keptnv2.GetSLITriggeredEventData{}
	if err := keptnv2.Decode(event.Data, eventData); err != nil {
		return nil, &sdk.Error{Err: err, StatusType: keptnv2.StatusErrored, ResultType: keptnv2.ResultFailed, Message: "failed to decode get-sli.triggered event: " + err.Error()}
	}

	// get prometheus API URL for the provided Project from Kubernetes Config Map
	prometheusAPIURL, err := getPrometheusAPIURL(eventData.Project, eh.kubeClient.CoreV1())
	if err != nil {
		return nil, &sdk.Error{Err: err, StatusType: keptnv2.StatusErrored, ResultType: keptnv2.ResultFailed, Message: "failed to get Prometheus API URL: " + err.Error()}
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
	projectCustomQueries, err := getCustomQueries(k.GetResourceHandler(), eventData.Project, eventData.Stage, eventData.Service)
	if err != nil {
		return nil, &sdk.Error{Err: err, StatusType: keptnv2.StatusErrored, ResultType: keptnv2.ResultFailed, Message: fmt.Sprintf("unable to retrieve custom queries for project %s: %e", eventData.Project, err)}
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
			Status:  keptnv2.StatusSucceeded,
			Result:  finalSLIEventResult,
			Project: eventData.Project,
			Stage:   eventData.Stage,
			Service: eventData.Service,
			Labels:  eventData.Labels,
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

	return getSliFinishedEventData, nil
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

func getCustomQueries(resourceHandler sdk.ResourceHandler, project string, stage string, service string) (map[string]string, error) {
	log.Println("Checking for custom SLI queries")

	customQueries, err := GetSLIConfiguration(resourceHandler, project, stage, service, utils.SliResourceURI)
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

// GetSLIConfiguration retrieves the SLI configuration for a service considering SLI configuration on stage and project level.
// First, the configuration of project-level is retrieved, which is then overridden by configuration on stage level,
// overridden by configuration on service level.
func GetSLIConfiguration(resourceHandler sdk.ResourceHandler, project string, stage string, service string, resourceURI string) (map[string]string, error) {
	var res *models.Resource
	var err error
	SLIs := make(map[string]string)

	// get sli config from project
	if project != "" {
		scope := api.NewResourceScope()
		scope.Project(project)
		scope.Resource(resourceURI)
		res, err = resourceHandler.GetResource(*scope)
		if err != nil {
			// return error except "resource not found" type
			if !strings.Contains(strings.ToLower(err.Error()), "resource not found") {
				return nil, err
			}
		}
		SLIs, err = addResourceContentToSLIMap(SLIs, res)
		if err != nil {
			return nil, err
		}
	}

	// get sli config from stage
	if project != "" && stage != "" {
		scope := api.NewResourceScope()
		scope.Project(project)
		scope.Stage(stage)
		scope.Resource(resourceURI)
		res, err = resourceHandler.GetResource(*scope)
		if err != nil {
			// return error except "resource not found" type
			if !strings.Contains(strings.ToLower(err.Error()), "resource not found") {
				return nil, err
			}
		}
		SLIs, err = addResourceContentToSLIMap(SLIs, res)
		if err != nil {
			return nil, err
		}
	}

	// get sli config from service
	if project != "" && stage != "" && service != "" {
		scope := api.NewResourceScope()
		scope.Project(project)
		scope.Stage(stage)
		scope.Service(service)
		scope.Resource(resourceURI)
		res, err = resourceHandler.GetResource(*scope)
		if err != nil {
			// return error except "resource not found" type
			if !strings.Contains(strings.ToLower(err.Error()), "resource not found") {
				return nil, err
			}
		}
		SLIs, err = addResourceContentToSLIMap(SLIs, res)
		if err != nil {
			return nil, err
		}
	}

	return SLIs, nil
}

func addResourceContentToSLIMap(SLIs map[string]string, resource *models.Resource) (map[string]string, error) {
	if resource != nil {
		sliConfig := keptncommon.SLIConfig{}
		err := yaml.Unmarshal([]byte(resource.ResourceContent), &sliConfig)
		if err != nil {
			return nil, err
		}

		for key, value := range sliConfig.Indicators {
			SLIs[key] = value
		}

		if len(SLIs) == 0 {
			return nil, errors.New("missing required field: indicators")
		}
	}
	return SLIs, nil
}
