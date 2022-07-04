package prometheus

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	alertConfig "github.com/prometheus/alertmanager/config"
	"github.com/prometheus/client_golang/api"
	apiv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"

	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	keptnv2 "github.com/keptn/go-utils/pkg/lib/v0_2_0"
)

const Throughput = "throughput"
const ErrorRate = "error_rate"
const RequestLatencyP50 = "response_time_p50"
const RequestLatencyP90 = "response_time_p90"
const RequestLatencyP95 = "response_time_p95"

// ErrInvalidData indicates that the retrieved data from the prometheus api is invalid
var /* const */ ErrInvalidData = errors.New("query did not return valid values")

// ErrNoValues indicates that no values where present in the prometheus api result
var /* const */ ErrNoValues = errors.New("query did not return any values")

// ErrMultipleValues indicates that multiple values where present in the prometheus api result
var /* const */ ErrMultipleValues = errors.New("query did return multiple values")

//go:generate mockgen -destination=fake/prometheusapi_mock.go -package=fake . API

// API is a type alias for the prometheus api interface
type API = apiv1.API

// Handler interacts with a prometheus API endpoint
type Handler struct {
	ApiURL         string
	Username       string
	Password       string
	Project        string
	Stage          string
	Service        string
	DeploymentType string
	Labels         map[string]string
	PrometheusAPI  API
	CustomFilters  []*keptnv2.SLIFilter
	CustomQueries  map[string]string
}

const alertManagerYamlTemplate = `global:
templates:
- '/etc/alertmanager/*.tmpl'
route:
  receiver: keptn_integration
  group_by: ['alertname', 'priority']
  group_wait: 10s
  repeat_interval: 30m
  routes:
    - receiver: keptn_integration
    # Send severity=webhook alerts to the webhook
      match:
        severity: webhook
      group_wait: 10s
      repeat_interval: 1m

receivers:
- name: keptn_integration
  webhook_configs:
  - url: http://prometheus-service.%s.svc.cluster.local:8080`

type PrometheusHelper struct {
	KubeAPI   *kubernetes.Clientset
	Namespace string
}

// NewPrometheusHelper creates a new PrometheusHelper
func NewPrometheusHelper(namespace string) (*PrometheusHelper, error) {

	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &PrometheusHelper{KubeAPI: clientSet, Namespace: namespace}, nil
}

func (p *PrometheusHelper) UpdateConfigMap(cm *v1.ConfigMap, namespace string) error {
	_, err := p.KubeAPI.CoreV1().ConfigMaps(namespace).Update(context.TODO(), cm, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	return nil
}

func (p *PrometheusHelper) GetConfigMap(name string, namespace string) (*v1.ConfigMap, error) {
	return p.KubeAPI.CoreV1().ConfigMaps(namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

func (p *PrometheusHelper) CreateConfigMap(cm *v1.ConfigMap, namespace string) error {
	_, err := p.KubeAPI.CoreV1().ConfigMaps(namespace).Create(context.TODO(), cm, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	return nil
}

func generateAlertManagerYaml(namespace string) string {
	return fmt.Sprintf(alertManagerYamlTemplate, namespace)
}

func (p *PrometheusHelper) UpdateAMConfigMap(name string, filename string, namespace string) error {
	getCM, err := p.GetConfigMap(name, namespace)
	if err != nil {
		return err
	}

	var config alertConfig.Config
	err = yaml.Unmarshal([]byte(getCM.Data[filename]), &config)
	if err != nil {
		return err
	}

	var keptnAlertConfig alertConfig.Config
	err = yaml.Unmarshal([]byte(generateAlertManagerYaml(p.Namespace)), &keptnAlertConfig)
	if err != nil {
		return err
	}

	// go over all receivers and check if keptn_integration is already there
	for _, rec := range config.Receivers {
		if rec.Name == "keptn_integration" {
			// already present, don't do anything
			return nil
		}
	}

	// go over all routes and check if keptn_integration is already there
	for _, route := range config.Route.Routes {
		if route.Receiver == "keptn_integration" {
			// already present, don't do anything
			return nil
		}
	}

	// insert keptn_integration in receivers and templates
	config.Receivers = append(config.Receivers, keptnAlertConfig.Receivers...)
	config.Templates = append(config.Templates, keptnAlertConfig.Templates...)
	config.Route.Routes = append(config.Route.Routes, keptnAlertConfig.Route.Routes...)
	getCM.Data[filename] = fmt.Sprint(config)

	return p.UpdateConfigMap(getCM, namespace)
}

// NewPrometheusHandler returns a new prometheus handler that interacts with the Prometheus REST API
func NewPrometheusHandler(apiURL string, eventData *keptnv2.EventData, deploymentType string, labels map[string]string, customFilters []*keptnv2.SLIFilter) *Handler {
	apiClient, _ := api.NewClient(api.Config{
		Address: apiURL,
	})

	v1api := apiv1.NewAPI(apiClient)

	ph := &Handler{
		ApiURL:         apiURL,
		Project:        eventData.Project,
		Stage:          eventData.Stage,
		Service:        eventData.Service,
		DeploymentType: deploymentType,
		Labels:         labels,
		PrometheusAPI:  v1api,
		CustomFilters:  customFilters,
	}

	return ph
}

// GetSLIValue retrieves the specified value via the Prometheus API
func (ph *Handler) GetSLIValue(metric string, start string, end string) (float64, error) {
	startUnix, err := parseUnixTimestamp(start)
	if err != nil {
		return 0, fmt.Errorf("unable to parse start timestamp: %w", err)
	}
	endUnix, _ := parseUnixTimestamp(end)
	if err != nil {
		return 0, fmt.Errorf("unable to parse end timestamp: %w", err)
	}
	query, err := ph.GetMetricQuery(metric, startUnix, endUnix)
	if err != nil {
		return 0, fmt.Errorf("unable to get metriy query: %w", err)
	}

	log.Println("GetSLIValue: Generated query: /api/v1/query?query=" + query + "&time=" + strconv.FormatInt(endUnix.Unix(), 10))

	result, w, err := ph.PrometheusAPI.Query(context.TODO(), query, endUnix)
	if err != nil {
		return 0, fmt.Errorf("unable to query prometheus api: %w", err)
	}

	if len(w) != 0 {
		log.Printf("Prometheus API returned warnings: %v", w)
	}

	// check if we can cast the result to a vector, it might be another data struct which we can't process
	resultVector, ok := result.(model.Vector)
	if !ok {
		return 0, fmt.Errorf("prometheus api response is not a Vector: %v", result)
	}

	// We are only allowed to return one value, if not the query may be malformed
	// we are using two different errors to give the user more information about the result
	if len(resultVector) == 0 {
		return 0, ErrNoValues
	} else if len(resultVector) > 1 {
		return 0, ErrMultipleValues
	}

	// parse the first entry as float and return the value if it's a valid float value
	resultValue := resultVector[0].Value.String()
	floatValue, err := strconv.ParseFloat(resultValue, 64)
	if err != nil || math.IsNaN(floatValue) {
		return 0, ErrInvalidData
	}

	log.Printf(fmt.Sprintf("Prometheus Result is %v\n", floatValue))
	return floatValue, nil
}

// GetMetricQuery returns the prometheus metric expression for the given SLI, start and end time
func (ph *Handler) GetMetricQuery(metric string, start time.Time, end time.Time) (string, error) {
	query := ph.CustomQueries[metric]
	if query != "" {
		query = ph.replaceQueryParameters(query, start, end)

		return query, nil
	}

	switch metric {
	case Throughput:
		return ph.getThroughputQuery(start, end), nil
	case ErrorRate:
		return ph.getErrorRateQuery(start, end), nil
	case RequestLatencyP50:
		return ph.getRequestLatencyQuery("50", start, end), nil
	case RequestLatencyP90:
		return ph.getRequestLatencyQuery("90", start, end), nil
	case RequestLatencyP95:
		return ph.getRequestLatencyQuery("95", start, end), nil
	default:
		return "", errors.New("unsupported SLI")
	}
}

func (ph *Handler) replaceQueryParameters(query string, start time.Time, end time.Time) string {
	for _, filter := range ph.CustomFilters {
		filter.Value = strings.Replace(filter.Value, "'", "", -1)
		filter.Value = strings.Replace(filter.Value, "\"", "", -1)
		query = strings.Replace(query, "$"+filter.Key, filter.Value, -1)
		query = strings.Replace(query, "$"+strings.ToUpper(filter.Key), filter.Value, -1)
	}
	query = strings.Replace(query, "$PROJECT", ph.Project, -1)
	query = strings.Replace(query, "$STAGE", ph.Stage, -1)
	query = strings.Replace(query, "$SERVICE", ph.Service, -1)
	query = strings.Replace(query, "$DEPLOYMENT", ph.DeploymentType, -1)

	// replace labels
	for key, value := range ph.Labels {
		query = strings.Replace(query, "$LABEL."+key, value, -1)
	}

	// replace duration
	durationString := strconv.FormatInt(getDurationInSeconds(start, end), 10) + "s"

	query = strings.Replace(query, "$DURATION_SECONDS", durationString, -1)
	return query
}

func (ph *Handler) getThroughputQuery(start time.Time, end time.Time) string {
	if ph.CustomQueries != nil && ph.CustomQueries["throughput"] != "" {
		query := ph.CustomQueries["throughput"]
		query = ph.replaceQueryParameters(query, start, end)
		return query
	}
	return ph.getDefaultThroughputQuery(start, end)
}

func (ph *Handler) getDefaultThroughputQuery(start time.Time, end time.Time) string {
	filterExpr := ph.getDefaultFilterExpression()
	durationString := strconv.FormatInt(getDurationInSeconds(start, end), 10) + "s"
	// e.g. sum(rate(http_requests_total{job="carts-sockshop-dev"}[30m]))&time=1571649085
	/*
		{
		    "status": "success",
		    "data": {
		        "resultType": "vector",
		        "result": [
		            {
		                "metric": {},
		                "value": [
		                    1571649085,
		                    "0.20111420612813372"
		                ]
		            }
		        ]
		    }
		}
	*/
	return "sum(rate(http_requests_total{" + filterExpr + "}[" + durationString + "]))"
}

func (ph *Handler) getErrorRateQuery(start time.Time, end time.Time) string {
	if ph.CustomQueries != nil && ph.CustomQueries["error_rate"] != "" {
		query := ph.CustomQueries["error_rate"]
		query = ph.replaceQueryParameters(query, start, end)
		return query
	}
	return ph.getDefaultErrorRateQuery(start, end)
}

func (ph *Handler) getDefaultErrorRateQuery(start time.Time, end time.Time) string {
	filterExpr := ph.getDefaultFilterExpression()
	durationString := strconv.FormatInt(getDurationInSeconds(start, end), 10) + "s"
	// e.g. sum(rate(http_requests_total{job="carts-sockshop-dev",status!~'2..'}[30m]))/sum(rate(http_requests_total{job="carts-sockshop-dev"}[30m]))&time=1571649085
	/*
		with value:
		{
		    "status": "success",
		    "data": {
		        "resultType": "vector",
		        "result": [
		            {
		                "metric": {},
		                "value": [
		                    1571649085,
		                    "1.00505917125441"
		                ]
		            }
		        ]
		    }
		}

		no value (error rate 0):
		{
		    "status": "success",
		    "data": {
		        "resultType": "vector",
		        "result": []
		    }
		}
	*/
	return "sum(rate(http_requests_total{" + filterExpr + ",status!~'2..'}[" + durationString + "]))/sum(rate(http_requests_total{" + filterExpr + "}[" + durationString + "]))"
}

func (ph *Handler) getRequestLatencyQuery(percentile string, start time.Time, end time.Time) string {
	if ph.CustomQueries != nil {
		query := ""
		switch percentile {
		case "50":
			query = ph.CustomQueries["response_time_p50"]
			break
		case "90":
			query = ph.CustomQueries["response_time_p90"]
			break
		case "95":
			query = ph.CustomQueries["response_time_p95"]
			break
		default:
			query = ""
		}
		if query != "" {
			query = ph.replaceQueryParameters(query, start, end)
			return query
		}
	}
	return ph.getDefaultRequestLatencyQuery(start, end, percentile)
}

func (ph *Handler) getDefaultRequestLatencyQuery(start time.Time, end time.Time, percentile string) string {
	filterExpr := ph.getDefaultFilterExpression()
	durationString := strconv.FormatInt(getDurationInSeconds(start, end), 10) + "s"
	// e.g. histogram_quantile(0.95, sum(rate(http_response_time_milliseconds_bucket{job='carts-sockshop-dev'}[30m])) by (le))&time=1571649085
	/*
		{
		    "status": "success",
		    "data": {
		        "resultType": "vector",
		        "result": [
		            {
		                "metric": {},
		                "value": [
		                    1571649085,
		                    "4.607481671642585"
		                ]
		            }
		        ]
		    }
		}
	*/
	return "histogram_quantile(0." + percentile + ",sum(rate(http_response_time_milliseconds_bucket{" + filterExpr + "}[" + durationString + "]))by(le))"
}

func (ph *Handler) getDefaultFilterExpression() string {
	filterExpression := ""
	jobFilterFound := false
	if ph.CustomFilters != nil && len(ph.CustomFilters) > 0 {
		for _, filter := range ph.CustomFilters {
			if filter.Key == "job" {
				jobFilterFound = true
			}
			/* if no operator has been included in the label filter, use exact matching (=), e.g.
			e.g.:
			key: handler
			value: ItemsController
			*/
			if !strings.HasPrefix(filter.Value, "=") && !strings.HasPrefix(filter.Value, "!=") && !strings.HasPrefix(filter.Value, "=~") && !strings.HasPrefix(filter.Value, "!~") {
				filter.Value = strings.Replace(filter.Value, "'", "", -1)
				filter.Value = strings.Replace(filter.Value, "\"", "", -1)
				if filterExpression != "" {
					filterExpression = filterExpression + "," + filter.Key + "='" + filter.Value + "'"
				} else {
					filterExpression = filter.Key + "='" + filter.Value + "'"
				}

			} else {
				/* if a valid operator (=, !=, =~, !~) is prepended to the value, use that one
				e.g.:
				key: handler
				value: !=HealthCheckController

				OR

				key: handler
				value: =~.+ItemsController|.+VersionController
				*/
				filter.Value = strings.Replace(filter.Value, "\"", "'", -1)
				if filterExpression != "" {
					filterExpression = filterExpression + "," + filter.Key + filter.Value
				} else {
					filterExpression = filter.Key + filter.Value
				}
			}
		}
	}
	if !jobFilterFound {
		if filterExpression != "" {
			filterExpression = "job='" + ph.Service + "-" + ph.Project + "-" + ph.Stage + "-canary'" + "," + filterExpression
		} else {
			filterExpression = "job='" + ph.Service + "-" + ph.Project + "-" + ph.Stage + "-canary'"
		}

	}
	return filterExpression
}

func parseUnixTimestamp(timestamp string) (time.Time, error) {
	parsedTime, err := time.Parse(time.RFC3339, timestamp)
	if err == nil {
		return parsedTime, nil
	}

	timestampInt, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return time.Now(), err
	}
	unix := time.Unix(timestampInt, 0)
	return unix, nil
}

func getDurationInSeconds(start, end time.Time) int64 {
	seconds := end.Sub(start).Seconds()
	return int64(math.Ceil(seconds))
}
