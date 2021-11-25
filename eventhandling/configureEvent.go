package eventhandling

import (
	"errors"
	"fmt"
	"strings"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/kelseyhightower/envconfig"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/keptn-contrib/prometheus-service/utils"

	configutils "github.com/keptn/go-utils/pkg/api/utils"
	keptnevents "github.com/keptn/go-utils/pkg/lib"
	"github.com/keptn/go-utils/pkg/lib/keptn"
	keptnv2 "github.com/keptn/go-utils/pkg/lib/v0_2_0"
	prometheus_model "github.com/prometheus/common/model"
	prometheusconfig "github.com/prometheus/prometheus/config"
	prometheus_sd_config "github.com/prometheus/prometheus/discovery/config"
	"github.com/prometheus/prometheus/discovery/targetgroup"
)

const Throughput = "throughput"
const ErrorRate = "error_rate"
const ResponseTimeP50 = "response_time_p50"
const ResponseTimeP90 = "response_time_p90"
const ResponseTimeP95 = "response_time_p95"
const metricsScrapePathEnvName = "METRICS_SCRAPE_PATH"
const environmentEnvName = "env"

const sliResourceURI = "prometheus/sli.yaml"

type envConfig struct {
	PrometheusNamespace           string `envconfig:"PROMETHEUS_NS" default:""`
	PrometheusConfigMap           string `envconfig:"PROMETHEUS_CM" default:""`
	PrometheusLabels              string `envconfig:"PROMETHEUS_LABELS" default:""`
	AlertManagerLabels            string `envconfig:"ALERT_MANAGER_LABELS" default:""`
	AlertManagerNamespace         string `envconfig:"ALERT_MANAGER_NS" default:""`
	AlertManagerConfigMap         string `envconfig:"ALERT_MANAGER_CM" default:""`
	AlertManagerTemplateConfigMap string `envconfig:"ALERT_MANAGER_TEMPLATE_CM" default:"alertmanager-templates"`
	PrometheusConfigFileName      string `envconfig:"PROMETHEUS_CONFIG_FILENAME" default:"prometheus.yml"`
	AlertManagerConfigFileName    string `envconfig:"ALERT_MANAGER_CONFIG_FILENAME" default:"alertmanager.yml"`
	ConfigurationServiceURL       string `envconfig:"CONFIGURATION_SERVICE" default:""`
}

// ConfigureMonitoringEventHandler is responsible for processing configure monitoring events
type ConfigureMonitoringEventHandler struct {
	logger       keptn.LoggerInterface
	event        cloudevents.Event
	keptnHandler *keptnv2.Keptn
}

type alertingRules struct {
	Groups []*alertingGroup `json:"groups" yaml:"groups"`
}

type alertingGroup struct {
	Name  string          `json:"name" yaml:"name"`
	Rules []*alertingRule `json:"rules" yaml:"rules"`
}

type alertingRule struct {
	Alert       string               `json:"alert" yaml:"alert"`
	Expr        string               `json:"expr" yaml:"expr"`
	For         string               `json:"for" yaml:"for"`
	Labels      *alertingLabel       `json:"labels" yaml:"labels"`
	Annotations *alertingAnnotations `json:"annotations" yaml:"annotations"`
}

type alertingLabel struct {
	Severity string `json:"severity" yaml:"severity"`
	PodName  string `json:"pod_name,omitempty" yaml:"pod_name"`
	Service  string `json:"service,omitempty" yaml:"service"`
	Stage    string `json:"stage,omitempty" yaml:"stage"`
	Project  string `json:"project,omitempty" yaml:"project"`
}

type alertingAnnotations struct {
	Summary     string `json:"summary" yaml:"summary"`
	Description string `json:"description" yaml:"descriptions"`
}

var env envConfig

// HandleEvent processes an event
func (eh ConfigureMonitoringEventHandler) HandleEvent() error {

	var shkeptncontext string
	_ = eh.event.Context.ExtensionAs("shkeptncontext", &shkeptncontext)

	eventData := &keptnevents.ConfigureMonitoringEventData{}
	if err := eh.event.DataAs(eventData); err != nil {
		return err
	}
	if eventData.Type != "prometheus" {
		return nil
	}

	err := eh.configurePrometheusAndStoreResources(eventData)
	if err != nil {
		eh.logger.Error(err.Error())
		return eh.handleError(eventData, err.Error())
	}

	if err = eh.sendConfigureMonitoringFinishedEvent(eventData, keptnv2.StatusSucceeded, keptnv2.ResultPass, "Prometheus successfully configured and rule created"); err != nil {
		eh.logger.Error(err.Error())
	}
	return nil
}

// configurePrometheusAndStoreResources
func (eh ConfigureMonitoringEventHandler) configurePrometheusAndStoreResources(eventData *keptnevents.ConfigureMonitoringEventData) error {

	if err := envconfig.Process("", &env); err != nil {
		eh.logger.Error("Failed to process env var: " + err.Error())
	}

	// (1) check if prometheus is installed
	if eh.isPrometheusInstalled() {
		eh.logger.Debug("Configure prometheus monitoring with keptn")
		if err := eh.updatePrometheusConfigMap(*eventData); err != nil {
			return err
		}

		eh.logger.Debug("Configure prometheus alert manager with keptn")
		err := eh.configurePrometheusAlertManager()
		if err != nil {
			return err
		}
	}

	// (2.1) delete prometheus pod
	if err := eh.deletePod(env.PrometheusLabels, env.PrometheusNamespace); err != nil {
		return err
	}

	// (2.1) delete prometheus alert manager pod
	if err := eh.deletePod(env.AlertManagerLabels, env.AlertManagerNamespace); err != nil {
		return err
	}

	return nil
}

func (eh ConfigureMonitoringEventHandler) deletePod(labels string, namespace string) error {
	eh.logger.Info("Deleting Pod with labels " + labels + "...")

	prometheusHelper, err := utils.NewPrometheusHelper()
	if err != nil {
		return err
	}

	labelArr := strings.Split(labels, ",")

	for _, label := range labelArr {
		err = prometheusHelper.DeletePod(label, namespace)
		if err != nil {
			return err
		}
	}

	eh.logger.Info("Deleting Pod successfully")

	return nil
}

func (eh ConfigureMonitoringEventHandler) isPrometheusInstalled() bool {
	eh.logger.Debug("Check if prometheus service in " + env.PrometheusNamespace + " namespace is available")
	config, err := rest.InClusterConfig()
	if err != nil {
		eh.logger.Debug(fmt.Sprintf("Could not initialize kubernetes client %s", err.Error()))
		return false
	}
	api, err := kubernetes.NewForConfig(config)

	if err != nil {
		eh.logger.Debug(fmt.Sprintf("Could not initialize kubernetes client %s", err.Error()))
		return false
	}

	svc, err := api.CoreV1().Services(env.PrometheusNamespace).List(metav1.ListOptions{
		LabelSelector: env.PrometheusLabels,
	})

	if err != nil {
		eh.logger.Debug(fmt.Sprintf("Prometheus service in %s namespace is not available. %s", env.PrometheusNamespace, err.Error()))
		return false
	}

	if len(svc.Items) > 0 {
		eh.logger.Debug("Prometheus service in " + env.PrometheusNamespace + " namespace is available")
		return true
	}

	return false
}

func (eh ConfigureMonitoringEventHandler) configurePrometheusAlertManager() error {
	eh.logger.Info("Configuring Prometheus AlertManager...")
	prometheusHelper, err := utils.NewPrometheusHelper()

	eh.logger.Info("Creating Prometheus AlertManager Template configmap...")
	err = prometheusHelper.CreateAMTempConfigMap(env.AlertManagerTemplateConfigMap, env.AlertManagerNamespace)
	if err != nil {
		return err
	}

	eh.logger.Info("Updating Prometheus AlertManager configmap...")
	err = prometheusHelper.UpdateAMConfigMap(env.AlertManagerConfigMap, env.AlertManagerConfigFileName, env.AlertManagerNamespace)
	if err != nil {
		return err
	}

	eh.logger.Info("Prometheus AlertManager configuration successfully")

	return nil
}

func (eh ConfigureMonitoringEventHandler) updatePrometheusConfigMap(eventData keptnevents.ConfigureMonitoringEventData) error {
	shipyard, err := eh.keptnHandler.GetShipyard()
	if err != nil {
		return err
	}

	api, err := getKubeClient()
	if err != nil {
		return err
	}

	cmPrometheus, err := api.CoreV1().ConfigMaps(env.PrometheusNamespace).Get(env.PrometheusConfigMap, metav1.GetOptions{})
	if err != nil {
		return err
	}
	config, err := prometheusconfig.Load(cmPrometheus.Data[env.PrometheusConfigFileName])
	if err != nil {
		return err
	}

	// check if alerting rules are already available
	var alertingRulesConfig alertingRules
	if cmPrometheus.Data["prometheus.rules"] != "" {
		yaml.Unmarshal([]byte(cmPrometheus.Data["prometheus.rules"]), &alertingRulesConfig)
	} else {
		alertingRulesConfig = alertingRules{}
	}
	// update
	for _, stage := range shipyard.Spec.Stages {
		var scrapeConfig *prometheusconfig.ScrapeConfig
		// (a) if a scrape config with the same name is available, update that one

		// <service>-primary.<project>-<stage>
		createScrapeJobConfig(scrapeConfig, config, eventData.Project, stage.Name, eventData.Service, false, true)
		// <service>-canary.<project>-<stage>
		createScrapeJobConfig(scrapeConfig, config, eventData.Project, stage.Name, eventData.Service, true, false)
		// <service>.<project>-<stage>
		createScrapeJobConfig(scrapeConfig, config, eventData.Project, stage.Name, eventData.Service, false, false)

		slos, err := retrieveSLOs(eventData, stage.Name, eh.logger)
		if err != nil || slos == nil {
			eh.logger.Info("No SLO file found for stage " + stage.Name + ". No alerting rules created for this stage")
			continue
		}

		// Create or update alerting group
		var alertingGroupConfig *alertingGroup
		alertingGroupName := eventData.Service + " " + eventData.Project + "-" + stage.Name + " alerts"
		alertingGroupConfig = getAlertingGroup(&alertingRulesConfig, alertingGroupName)
		if alertingGroupConfig == nil {
			alertingGroupConfig = &alertingGroup{
				Name: alertingGroupName,
			}
			alertingRulesConfig.Groups = append(alertingRulesConfig.Groups, alertingGroupConfig)
		}

		for _, objective := range slos.Objectives {

			expr, err := eh.getSLIQuery(eventData.Project, stage.Name, eventData.Service, objective.SLI, slos.Filter)
			if err != nil || expr == "" {
				eh.logger.Error("No query defined for SLI " + objective.SLI + " in project " + eventData.Project)
				continue
			}

			if objective.Pass != nil {
				for _, criteriaGroup := range objective.Pass {
					for _, criteria := range criteriaGroup.Criteria {
						if strings.Contains(criteria, "+") || strings.Contains(criteria, "-") || strings.Contains(criteria, "%") || (!strings.Contains(criteria, "<") && !strings.Contains(criteria, ">")) {
							continue
						}
						criteriaString := strings.Replace(criteria, "=", "", -1)
						if strings.Contains(criteriaString, "<") {
							criteriaString = strings.Replace(criteriaString, "<", ">", -1)
						} else {
							criteriaString = strings.Replace(criteriaString, ">", "<", -1)
						}

						var newAlertingRule *alertingRule
						ruleName := objective.SLI
						newAlertingRule = getAlertingRuleOfGroup(alertingGroupConfig, ruleName)
						if newAlertingRule == nil {
							newAlertingRule = &alertingRule{
								Alert: ruleName,
							}
							alertingGroupConfig.Rules = append(alertingGroupConfig.Rules, newAlertingRule)
						}
						newAlertingRule.Alert = ruleName
						newAlertingRule.Expr = expr + criteriaString
						newAlertingRule.For = "10m" // TODO: introduce alert duration concept in SLO?
						newAlertingRule.Labels = &alertingLabel{
							Severity: "webhook",
							PodName:  eventData.Service + "-primary",
							Service:  eventData.Service,
							Project:  eventData.Project,
							Stage:    stage.Name,
						}
						newAlertingRule.Annotations = &alertingAnnotations{
							Summary:     ruleName,
							Description: "Pod name {{ $labels.pod_name }}",
						}
					}
				}
			}
		}
	}
	alertingRulesYAMLString, err := yaml.Marshal(alertingRulesConfig)
	if err != nil {
		return err
	}
	// apply
	cmPrometheus.Data["alerting_rules.yml"] = string(alertingRulesYAMLString)
	cmPrometheus.Data[env.PrometheusConfigFileName] = config.String()
	_, err = api.CoreV1().ConfigMaps(env.PrometheusNamespace).Update(cmPrometheus)
	if err != nil {
		return err
	}
	return nil
}

func getKubeClient() (*kubernetes.Clientset, error) {
	k8sConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	api, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return nil, err
	}
	return api, nil
}

func getDefaultFilterExpression(project string, stage string, service string, filters map[string]string) string {
	filterExpression := "job='" + service + "-" + project + "-" + stage + "-primary'"
	if filters != nil && len(filters) > 0 {
		for key, value := range filters {
			/* if no operator has been included in the label filter, use exact matching (=), e.g.
			e.g.:
			key: handler
			value: ItemsController
			*/
			sanitizedValue := value
			if !strings.HasPrefix(sanitizedValue, "=") && !strings.HasPrefix(sanitizedValue, "!=") && !strings.HasPrefix(sanitizedValue, "=~") && !strings.HasPrefix(sanitizedValue, "!~") {
				sanitizedValue = strings.Replace(sanitizedValue, "'", "", -1)
				sanitizedValue = strings.Replace(sanitizedValue, "\"", "", -1)
				filterExpression = filterExpression + "," + key + "='" + sanitizedValue + "'"
			} else {
				/* if a valid operator (=, !=, =~, !~) is prepended to the value, use that one
				e.g.:
				key: handler
				value: !=HealthCheckController

				OR

				key: handler
				value: =~.+ItemsController|.+VersionController
				*/
				sanitizedValue = strings.Replace(sanitizedValue, "\"", "'", -1)
				filterExpression = filterExpression + "," + key + sanitizedValue
			}
		}
	}
	return filterExpression
}

func (eh ConfigureMonitoringEventHandler) getSLIQuery(project string, stage string, service string, sli string, filters map[string]string) (string, error) {
	query, err := eh.getCustomQuery(project, stage, service, sli)
	if err == nil && query != "" {
		query = replaceQueryParameters(query, project, stage, service, filters)

		return query, nil
	}
	switch sli {
	case Throughput:
		eh.logger.Info("Using default query for throughput")
		query = getDefaultThroughputQuery(project, stage, service, filters)
	case ErrorRate:
		eh.logger.Info("Using default query for error_rate")
		query = getDefaultErrorRateQuery(project, stage, service, filters)
	case ResponseTimeP50:
		eh.logger.Info("Using default query for response_time_p50")
		query = getDefaultResponseTimeQuery(project, stage, service, filters, "50")
	case ResponseTimeP90:
		eh.logger.Info("Using default query for response_time_p90")
		query = getDefaultResponseTimeQuery(project, stage, service, filters, "90")
	case ResponseTimeP95:
		eh.logger.Info("Using default query for response_time_p95")
		query = getDefaultResponseTimeQuery(project, stage, service, filters, "95")
	default:
		return "", errors.New("unsupported SLI")
	}
	query = replaceQueryParameters(query, project, stage, service, filters)
	return query, nil
}

func getDefaultThroughputQuery(project string, stage string, service string, filters map[string]string) string {
	filterExpr := getDefaultFilterExpression(project, stage, service, filters)
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
	return "sum(rate(http_requests_total{" + filterExpr + "}[180s]))"
}

func getDefaultErrorRateQuery(project string, stage string, service string, filters map[string]string) string {
	filterExpr := getDefaultFilterExpression(project, stage, service, filters)
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
	return "sum(rate(http_requests_total{" + filterExpr + ",status!~'2..'}[180s]))/sum(rate(http_requests_total{" + filterExpr + "}[180s]))"
}

func getDefaultResponseTimeQuery(project string, stage string, service string, filters map[string]string, percentile string) string {
	filterExpr := getDefaultFilterExpression(project, stage, service, filters)
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
	return "histogram_quantile(0." + percentile + ",sum(rate(http_response_time_milliseconds_bucket{" + filterExpr + "}[180s]))by(le))"
}

func replaceQueryParameters(query string, project string, stage string, service string, filters map[string]string) string {
	for key, value := range filters {
		sanitizedValue := value
		sanitizedValue = strings.Replace(sanitizedValue, "'", "", -1)
		sanitizedValue = strings.Replace(sanitizedValue, "\"", "", -1)
		query = strings.Replace(query, "$"+key, sanitizedValue, -1)
		query = strings.Replace(query, "$"+strings.ToUpper(key), sanitizedValue, -1)
	}
	query = strings.Replace(query, "$PROJECT", project, -1)
	query = strings.Replace(query, "$STAGE", stage, -1)
	query = strings.Replace(query, "$SERVICE", service, -1)
	query = strings.Replace(query, "$project", project, -1)
	query = strings.Replace(query, "$stage", stage, -1)
	query = strings.Replace(query, "$service", service, -1)
	query = strings.Replace(query, "$DURATION_SECONDS", "180s", -1)
	return query
}

func (eh ConfigureMonitoringEventHandler) getCustomQuery(project, stage, service string, sli string) (string, error) {

	customQueries, err := eh.keptnHandler.GetSLIConfiguration(project, stage, service, sliResourceURI)

	if err != nil {
		return "", err
	}

	return customQueries[sli], nil
}

func (eh ConfigureMonitoringEventHandler) extractCustomQueryFromCM(configMap *v1.ConfigMap, sli string, project string) (string, error) {
	if configMap == nil || configMap.Data == nil || configMap.Data["custom-queries"] == "" {
		eh.logger.Info("No custom query defined for SLI " + sli + " in project " + project)
		return "", nil
	}
	customQueries := make(map[string]string)
	err := yaml.Unmarshal([]byte(configMap.Data["custom-queries"]), &customQueries)
	if err != nil || customQueries == nil || customQueries[sli] == "" {
		eh.logger.Info("No custom query defined for SLI " + sli + " in project " + project)
		return "", nil
	}
	query := customQueries[sli]
	return query, nil
}

func createScrapeJobConfig(scrapeConfig *prometheusconfig.ScrapeConfig, config *prometheusconfig.Config, project string, stage string, service string, isCanary bool, isPrimary bool) {
	scrapeConfigName := service + "-" + project + "-" + stage
	var scrapeEndpoint string
	if isCanary {
		scrapeConfigName = scrapeConfigName + "-canary"
		scrapeEndpoint = service + "-canary." + project + "-" + stage + ":80"
	} else if isPrimary {
		scrapeConfigName = scrapeConfigName + "-primary"
		scrapeEndpoint = service + "-primary." + project + "-" + stage + ":80"
	} else {
		scrapeEndpoint = service + "." + project + "-" + stage + ":80"
	}

	scrapeConfig = getScrapeConfig(config, scrapeConfigName)
	// (b) if not, create a new scrape config
	if scrapeConfig == nil {
		scrapeConfig = &prometheusconfig.ScrapeConfig{}
		config.ScrapeConfigs = append(config.ScrapeConfigs, scrapeConfig)
	}

	// define scrape job name
	scrapeConfig.JobName = scrapeConfigName
	// set scrape interval to 5 seconds
	scrapeConfig.ScrapeInterval = prometheus_model.Duration(5 * time.Second)
	// configure metrics path (default: /metrics)
	scrapeConfig.MetricsPath = utils.EnvVarOrDefault(metricsScrapePathEnvName, "/metrics")
	scrapeConfig.ServiceDiscoveryConfig = prometheus_sd_config.ServiceDiscoveryConfig{
		StaticConfigs: []*targetgroup.Group{
			{
				Targets: []prometheus_model.LabelSet{
					{prometheus_model.AddressLabel: prometheus_model.LabelValue(scrapeEndpoint)},
				},
			},
		},
	}
}

func getAlertingRuleOfGroup(alertingGroup *alertingGroup, alertName string) *alertingRule {
	for _, rule := range alertingGroup.Rules {
		if rule.Alert == alertName {
			return rule
		}
	}
	return nil
}

func getAlertingGroup(alertingRulesConfig *alertingRules, groupName string) *alertingGroup {
	for _, alertingGroup := range alertingRulesConfig.Groups {
		if alertingGroup.Name == groupName {
			return alertingGroup
		}
	}
	return nil
}

func getScrapeConfig(config *prometheusconfig.Config, name string) *prometheusconfig.ScrapeConfig {
	for _, scrapeConfig := range config.ScrapeConfigs {
		if scrapeConfig.JobName == name {
			return scrapeConfig
		}
	}
	return nil
}

func getConfigurationServiceURL() string {
	return env.ConfigurationServiceURL
}

func retrieveSLOs(eventData keptnevents.ConfigureMonitoringEventData, stage string, logger keptn.LoggerInterface) (*keptnevents.ServiceLevelObjectives, error) {
	resourceHandler := configutils.NewResourceHandler(getConfigurationServiceURL())

	resource, err := resourceHandler.GetServiceResource(eventData.Project, stage, eventData.Service, "slo.yaml")
	if err != nil || resource.ResourceContent == "" {
		return nil, errors.New("No SLO file available for service " + eventData.Service + " in stage " + stage)
	}
	var slos keptnevents.ServiceLevelObjectives

	err = yaml.Unmarshal([]byte(resource.ResourceContent), &slos)

	if err != nil {
		return nil, errors.New("Invalid SLO file format")
	}

	return &slos, nil
}

func (eh ConfigureMonitoringEventHandler) sendConfigureMonitoringFinishedEvent(configureMonitoringData *keptnevents.ConfigureMonitoringEventData, status keptnv2.StatusType, result keptnv2.ResultType, msg string) error {
	cmFinishedEvent := &keptnv2.ConfigureMonitoringFinishedEventData{
		EventData: keptnv2.EventData{
			Project: configureMonitoringData.Project,
			Service: configureMonitoringData.Service,
			Status:  status,
			Result:  result,
			Message: msg,
		},
	}
	keptnContext, _ := eh.event.Context.GetExtension("shkeptncontext")
	triggeredID := eh.event.Context.GetID()

	event := cloudevents.NewEvent()
	event.SetSource("prometheus-service")
	event.SetDataContentType(cloudevents.ApplicationJSON)
	event.SetType(keptnv2.GetFinishedEventType(keptnv2.ConfigureMonitoringTaskName))
	event.SetData(cloudevents.ApplicationJSON, cmFinishedEvent)
	event.SetExtension("shkeptncontext", keptnContext)
	event.SetExtension("triggeredid", triggeredID)

	if err := eh.keptnHandler.SendCloudEvent(event); err != nil {
		return fmt.Errorf("could not send %s event: %s", keptnv2.GetFinishedEventType(keptnv2.ConfigureMonitoringTaskName), err.Error())
	}

	return nil
}

func (eh ConfigureMonitoringEventHandler) handleError(e *keptnevents.ConfigureMonitoringEventData, msg string) error {
	//logger.Error(msg)
	if err := eh.sendConfigureMonitoringFinishedEvent(e, keptnv2.StatusErrored, keptnv2.ResultFailed, msg); err != nil {
		eh.logger.Error(err.Error())
	}
	return errors.New(msg)
}
