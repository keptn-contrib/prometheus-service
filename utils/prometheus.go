package utils

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	keptncommon "github.com/keptn/go-utils/pkg/lib/keptn"
	alertConfig "github.com/prometheus/alertmanager/config"
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

type prometheusResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric struct {
			} `json:"metric"`
			Value []interface{} `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

// Handler interacts with a prometheus API endpoint
type Handler struct {
	ApiURL        string
	Username      string
	Password      string
	Project       string
	Stage         string
	Service       string
	HTTPClient    *http.Client
	CustomFilters []*keptnv2.SLIFilter
	CustomQueries map[string]string
}

const alertManagerYml = `global:
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
  - url: http://prometheus-service.keptn.svc.cluster.local:8080`

type PrometheusHelper struct {
	KubeApi *kubernetes.Clientset
}

// NewPrometheusHelper creates a new PrometheusHelper
func NewPrometheusHelper() (*PrometheusHelper, error) {

	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	clientSet, err := kubernetes.NewForConfig(config)

	if err != nil {
		return nil, err
	}

	return &PrometheusHelper{KubeApi: clientSet}, nil
}

func (p *PrometheusHelper) UpdateConfigMap(cm *v1.ConfigMap, namespace string) error {
	_, err := p.KubeApi.CoreV1().ConfigMaps(namespace).Update(cm)
	if err != nil {
		return err
	}

	return nil
}

func (p *PrometheusHelper) GetConfigMap(name string, namespace string) (*v1.ConfigMap, error) {
	return p.KubeApi.CoreV1().ConfigMaps(namespace).Get(name, metav1.GetOptions{})
}

func (p *PrometheusHelper) CreateConfigMap(cm *v1.ConfigMap, namespace string) error {
	_, err := p.KubeApi.CoreV1().ConfigMaps(namespace).Create(cm)
	if err != nil {
		return err
	}

	return nil
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
	err = yaml.Unmarshal([]byte(alertManagerYml), &keptnAlertConfig)
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
func NewPrometheusHandler(apiURL string, project string, stage string, service string, customFilters []*keptnv2.SLIFilter) *Handler {
	ph := &Handler{
		ApiURL:        apiURL,
		Project:       project,
		Stage:         stage,
		Service:       service,
		HTTPClient:    &http.Client{},
		CustomFilters: customFilters,
	}

	return ph
}

// GetSLIValue retrieves the specified value via the Prometheus API
func (ph *Handler) GetSLIValue(metric string, start string, end string, logger keptncommon.LoggerInterface) (float64, error) {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	startUnix, err := parseUnixTimestamp(start)
	if err != nil {
		return 0, err
	}
	endUnix, _ := parseUnixTimestamp(end)
	if err != nil {
		return 0, err
	}
	query, err := ph.getMetricQuery(metric, startUnix, endUnix)
	if err != nil {
		return 0, err
	}
	queryString := ph.ApiURL + "/api/v1/query?query=" + url.QueryEscape(query) + "&time=" + strconv.FormatInt(endUnix.Unix(), 10)
	logger.Info("Generated query: /api/v1/query?query=" + query + "&time=" + strconv.FormatInt(endUnix.Unix(), 10))

	req, err := http.NewRequest("GET", queryString, nil)
	req.Header.Set("Content-Type", "application/json")

	resp, err := ph.HTTPClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return 0, errors.New("metric could not be received")
	}

	prometheusResult := &prometheusResponse{}

	err = json.Unmarshal(body, prometheusResult)
	if err != nil {
		return 0, err
	}

	if len(prometheusResult.Data.Result) == 0 || len(prometheusResult.Data.Result[0].Value) == 0 {
		logger.Info("Prometheus Result is 0, returning value 0")
		// for the error rate query, the result is received with no value if the error rate is 0, so we have to assume that's OK at this point
		return 0, nil
	}

	parsedValue := fmt.Sprintf("%v", prometheusResult.Data.Result[0].Value[1])
	floatValue, err := strconv.ParseFloat(parsedValue, 64)
	logger.Info(fmt.Sprintf("Prometheus Result is %v", floatValue))
	if err != nil {
		return 0, nil
	}
	return floatValue, nil
}

func (ph *Handler) getMetricQuery(metric string, start time.Time, end time.Time) (string, error) {
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
	query = strings.Replace(query, "$project", ph.Project, -1)
	query = strings.Replace(query, "$stage", ph.Stage, -1)
	query = strings.Replace(query, "$service", ph.Service, -1)
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
