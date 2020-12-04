package eventhandling

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"gopkg.in/yaml.v2"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"net/url"
	"strings"

	kubeutils "github.com/keptn/kubernetes-utils/pkg"

	"github.com/keptn-contrib/prometheus-service/utils"

	"github.com/keptn/go-utils/pkg/api/models"
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
const configservice = "CONFIGURATION_SERVICE"
const api = "API"
const keptnPrometheusSLIConfigMapName = "prometheus-sli-config"
const podNamespaceEnvName = "POD_NAMESPACE"
const metricsScrapePathEnvName = "METRICS_SCRAPE_PATH"
const environmentEnvName = "env"

type doneEventData struct {
	Result  string `json:"result"`
	Message string `json:"message"`
	Version string `json:"version"`
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

// GotEvent is the event handler of cloud events
func GotEvent(ctx context.Context, event cloudevents.Event) error {
	var shkeptncontext string
	_ = event.Context.ExtensionAs("shkeptncontext", &shkeptncontext)

	// process event
	if event.Type() == keptnevents.ConfigureMonitoringEventType {
		eventData := &keptnevents.ConfigureMonitoringEventData{}
		if err := event.DataAs(eventData); err != nil {
			return err
		}
		if eventData.Type != "prometheus" {
			return nil
		}

		stdLogger := keptn.NewLogger(shkeptncontext, event.Context.GetID(), "prometheus-service")

		var logger keptn.LoggerInterface

		connData := &keptn.ConnectionData{}
		if err := event.DataAs(connData); err != nil ||
			*connData.EventContext.KeptnContext == "" || *connData.EventContext.Token == "" {
			logger = stdLogger
			logger.Debug("No Websocket connection data available")
		} else {
			apiServiceURL, err := utils.GetServiceEndpoint(api)
			if err != nil {
				logger.Error(err.Error())
				return nil
			}
			ws, _, err := keptn.OpenWS(*connData, apiServiceURL)
			defer ws.Close()
			if err != nil {
				stdLogger.Error(fmt.Sprintf("Opening websocket connection failed. %s", err.Error()))
				return nil
			}
			combinedLogger := keptn.NewCombinedLogger(stdLogger, ws, shkeptncontext)
			defer combinedLogger.Terminate("")
			logger = combinedLogger
		}

		eventBrokerURL, err := utils.GetEventBrokerURL()
		if err != nil {
			logger.Error(err.Error())
			return err
		}
		keptnHandler, err := keptnv2.NewKeptn(&event, keptn.KeptnOpts{
			EventBrokerURL: eventBrokerURL,
		})
		if err != nil {
			logger.Error("Could not initialize Keptn handler: " + err.Error())
			return err
		}
		keptnHandler.Logger = logger

		version, err := configurePrometheusAndStoreResources(eventData, logger, keptnHandler)
		if err := logErrAndRespondWithDoneEvent(event, version, err, keptnHandler); err != nil {
			return err
		}

		return nil
	}

	const errorMsg = "Received unexpected keptn event that cannot be processed"
	// if err := websocketutil.WriteWSLog(ws, createEventCopy(event, "sh.keptn.events.log"), errorMsg, true, "INFO"); err != nil {
	// 	logger.Error(fmt.Sprintf("Could not write log to websocket. %s", err.Error()))
	// }
	return errors.New(errorMsg)
}

// configurePrometheusAndStoreResources
func configurePrometheusAndStoreResources(eventData *keptnevents.ConfigureMonitoringEventData, logger keptn.LoggerInterface, keptnHandler *keptnv2.Keptn) (*models.Version, error) {
	// (1) check if prometheus is installed, otherwise install prometheus and alert manager
	if !isPrometheusInstalled(logger) {
		logger.Debug("Installing prometheus monitoring")
		err := installPrometheus(logger)
		if err != nil {
			return nil, err
		}

		logger.Debug("Installing prometheus alert manager")
		err = installPrometheusAlertManager(logger)
		if err != nil {
			return nil, err
		}
	}
	fmt.Println("prometheus is installed, updating config maps")

	// (2) update config map with alert rule
	if err := updatePrometheusConfigMap(*eventData, logger, keptnHandler); err != nil {
		return nil, err
	}

	// (2.1) delete prometheus pod
	err := deletePrometheusPod()
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func deletePrometheusPod() error {

	if err := kubeutils.RestartPodsWithSelector(utils.EnvVarEqualsTo(environmentEnvName, "production"), "monitoring", "app=prometheus-server"); err != nil {
		return err
	}
	return nil
}

func isPrometheusInstalled(logger keptn.LoggerInterface) bool {
	logger.Debug("Check if prometheus service in monitoring namespace is available")
	config, err := rest.InClusterConfig()
	if err != nil {
		logger.Debug(fmt.Sprintf("Could not initialize kubernetes client %s", err.Error()))
		return false
	}
	api, err := kubernetes.NewForConfig(config)

	if err != nil {
		logger.Debug(fmt.Sprintf("Could not initialize kubernetes client %s", err.Error()))
		return false
	}

	_, err = api.CoreV1().Services("monitoring").Get("prometheus-service", metav1.GetOptions{})
	if err != nil {
		logger.Debug(fmt.Sprintf("Prometheus service in monitoring namespace is not available. %s", err.Error()))
		return false
	}

	logger.Debug("Prometheus service in monitoring namespace is available")
	return true
}

func installPrometheus(logger keptn.LoggerInterface) error {
	logger.Info("Installing Prometheus...")
	prometheusHelper, err := utils.NewPrometheusHelper()
	if err != nil {
		logger.Debug(fmt.Sprintf("Could not initialize kubernetes client %s", err.Error()))
		return err
	}
	logger.Debug("Apply namespace for prometheus monitoring")
	err = prometheusHelper.CreateOrUpdatePrometheusNamespace()
	if err != nil {
		return err
	}

	//config-map.yaml
	logger.Debug("Apply config map for prometheus monitoring")
	err = prometheusHelper.CreateOrUpdatePrometheusConfigMap()
	if err != nil {
		return err
	}

	//cluster-role.yaml
	logger.Debug("Apply cluster role for prometheus monitoring")
	err = prometheusHelper.CreateOrUpdatePrometheusClusterRole()
	if err != nil {
		return err
	}

	//prometheus.yaml
	logger.Debug("Apply service and deployment for prometheus monitoring")
	err = prometheusHelper.CreateOrUpdatePrometheusDeployment()
	if err != nil {
		return err
	}

	logger.Info("Prometheus installed successfully")

	return nil
}

func installPrometheusAlertManager(logger keptn.LoggerInterface) error {
	logger.Info("Installing Prometheus AlertManager...")
	prometheusHelper, err := utils.NewPrometheusHelper()
	//alertmanager-configmap.yaml
	logger.Debug("Apply configmap for prometheus alert manager")
	err = prometheusHelper.CreateOrUpdateAlertManagerConfigMap()
	if err != nil {
		return err
	}

	//alertmanager-template.yaml
	logger.Debug("Apply configmap template for prometheus alert manager")
	err = prometheusHelper.CreateOrUpdateAlertManagerTemplatesConfigMap()
	if err != nil {
		return err
	}

	//alertmanager-deployment.yaml
	logger.Debug("Apply deployment for prometheus alert manager")
	err = prometheusHelper.CreateOrUpdateAlertManagerDeployment()
	if err != nil {
		return err
	}

	//alertmanager-svc.yaml
	logger.Debug("Apply service for prometheus alert manager")
	err = prometheusHelper.CreateOrUpdateAlertManagerService()
	if err != nil {
		return err
	}

	logger.Info("Prometheus AlertManager installed successfully")

	return nil
}

func updatePrometheusConfigMap(eventData keptnevents.ConfigureMonitoringEventData, logger keptn.LoggerInterface, keptnHandler *keptnv2.Keptn) error {
	shipyard, err := keptnHandler.GetShipyard()
	if err != nil {
		return err
	}

	api, err := getKubeClient()
	if err != nil {
		return err
	}

	cmPrometheus, err := api.CoreV1().ConfigMaps("monitoring").Get("prometheus-server-conf", metav1.GetOptions{})
	if err != nil {
		return err
	}
	config, err := prometheusconfig.Load(cmPrometheus.Data["prometheus.yml"])
	if err != nil {
		return err
	}
	fmt.Println(config)

	// check if alerting rules are already availablre
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

		slos, err := retrieveSLOs(eventData, stage.Name, logger)
		if err != nil || slos == nil {
			logger.Info("No SLO file found for stage " + stage.Name + ". No alerting rules created for this stage")
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

			expr, err := getSLIQuery(eventData.Project, stage.Name, eventData.Service, objective.SLI, slos.Filter, logger)
			if err != nil || expr == "" {
				logger.Error("No query defined for SLI " + objective.SLI + " in project " + eventData.Project)
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
	cmPrometheus.Data["prometheus.rules"] = string(alertingRulesYAMLString)
	cmPrometheus.Data["prometheus.yml"] = config.String()
	_, err = api.CoreV1().ConfigMaps("monitoring").Update(cmPrometheus)
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
	filterExpression := "job='" + service + "-" + project + "-" + stage + "'"
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

func getSLIQuery(project string, stage string, service string, sli string, filters map[string]string, logger keptn.LoggerInterface) (string, error) {
	query, err := getCustomQuery(project, sli, logger)
	if err == nil && query != "" {
		query = replaceQueryParameters(query, project, stage, service, filters)

		return query, nil
	}
	switch sli {
	case Throughput:
		logger.Info("Using default query for throughput")
		query = getDefaultThroughputQuery(project, stage, service, filters)
	case ErrorRate:
		logger.Info("Using default query for error_rate")
		query = getDefaultErrorRateQuery(project, stage, service, filters)
	case ResponseTimeP50:
		logger.Info("Using default query for response_time_p50")
		query = getDefaultResponseTimeQuery(project, stage, service, filters, "50")
	case ResponseTimeP90:
		logger.Info("Using default query for response_time_p90")
		query = getDefaultResponseTimeQuery(project, stage, service, filters, "90")
	case ResponseTimeP95:
		logger.Info("Using default query for response_time_p95")
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

func getCustomQuery(project string, sli string, logger keptn.LoggerInterface) (string, error) {
	kubeClient, err := getKubeClient()
	if err != nil {
		logger.Error("could not create kube client")
		return "", errors.New("could not create kube client")
	}
	logger.Info("Checking for custom SLI queries for project " + project)

	// try to get project-specific configMap
	configMap, err := kubeClient.CoreV1().ConfigMaps(utils.EnvVar(podNamespaceEnvName)).Get(keptnPrometheusSLIConfigMapName+"-"+project, metav1.GetOptions{})

	if err == nil {
		query, err := extractCustomQueryFromCM(configMap, logger, sli, project)
		if err == nil && query != "" {
			return query, nil
		}
	}

	// if no config Map could be found, try to get the global one
	configMap, err = kubeClient.CoreV1().ConfigMaps(utils.EnvVar(podNamespaceEnvName)).Get(keptnPrometheusSLIConfigMapName, metav1.GetOptions{})

	query, err := extractCustomQueryFromCM(configMap, logger, sli, project)
	if err != nil {
		return "", err
	}

	return query, nil

}

func extractCustomQueryFromCM(configMap *v1.ConfigMap, logger keptn.LoggerInterface, sli string, project string) (string, error) {
	if configMap == nil || configMap.Data == nil || configMap.Data["custom-queries"] == "" {
		logger.Info("No custom query defined for SLI " + sli + " in project " + project)
		return "", nil
	}
	customQueries := make(map[string]string)
	err := yaml.Unmarshal([]byte(configMap.Data["custom-queries"]), &customQueries)
	if err != nil || customQueries == nil || customQueries[sli] == "" {
		logger.Info("No custom query defined for SLI " + sli + " in project " + project)
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
	scrapeConfig.JobName = scrapeConfigName
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
	if utils.EnvVarEqualsTo(environmentEnvName, "production") {
		return "configuration-service.keptn.svc.cluster.local:8080"
	}
	return "localhost:6060"
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

// logErrAndRespondWithDoneEvent sends a keptn done event to the keptn eventbroker
func logErrAndRespondWithDoneEvent(event cloudevents.Event, version *models.Version, err error, keptnHandler *keptnv2.Keptn) error {
	var result = "success"
	//var webSocketMessage = "Prometheus successfully configured"
	var eventMessage = "Prometheus successfully configured and rule created"

	if err != nil { // error
		result = "error"
		eventMessage = fmt.Sprintf("%s.", err.Error())
		//webSocketMessage = eventMessage
		keptnHandler.Logger.Error(eventMessage)
	} else { // success
		keptnHandler.Logger.Info(eventMessage)
	}

	// if err := websocketutil.WriteWSLog(ws, createEventCopy(event, "sh.keptn.events.log"), webSocketMessage, true, "INFO"); err != nil {
	// 	logger.Error(fmt.Sprintf("Could not write log to websocket. %s", err.Error()))
	// }
	if err := sendDoneEvent(event, result, eventMessage, version, keptnHandler); err != nil {
		keptnHandler.Logger.Error(fmt.Sprintf("No sh.keptn.event.done event sent. %s", err.Error()))
	}

	return err
}

// createEventCopy creates a deep copy of a CloudEvent
func createEventCopy(eventSource cloudevents.Event, eventType string) cloudevents.Event {
	var shkeptncontext string
	eventSource.Context.ExtensionAs("shkeptncontext", &shkeptncontext)
	var shkeptnphaseid string
	eventSource.Context.ExtensionAs("shkeptnphaseid", &shkeptnphaseid)
	var shkeptnphase string
	eventSource.Context.ExtensionAs("shkeptnphase", &shkeptnphase)
	var shkeptnstepid string
	eventSource.Context.ExtensionAs("shkeptnstepid", &shkeptnstepid)
	var shkeptnstep string
	eventSource.Context.ExtensionAs("shkeptnstep", &shkeptnstep)

	source, _ := url.Parse("prometheus-service")

	event := cloudevents.NewEvent()
	event.SetID(uuid.New().String())
	event.SetTime(time.Now())
	event.SetType(eventType)
	event.SetSource(source.String())
	event.SetExtension("shkeptncontext", shkeptncontext)
	event.SetExtension("shkeptnphaseid", shkeptnphaseid)
	event.SetExtension("shkeptnphase", shkeptnphase)
	event.SetExtension("shkeptnstepid", shkeptnstepid)
	event.SetExtension("shkeptnstep", shkeptnstep)
	event.SetDataContentType(cloudevents.ApplicationJSON)

	return event
}

// sendDoneEvent prepares a keptn done event and sends it to the eventbroker
func sendDoneEvent(receivedEvent cloudevents.Event, result string, message string, version *models.Version, keptnHandler *keptnv2.Keptn) error {

	doneEvent := createEventCopy(receivedEvent, "sh.keptn.events.done")

	eventData := doneEventData{
		Result:  result,
		Message: message,
	}

	if version != nil {
		eventData.Version = version.Version
	}

	doneEvent.SetData(cloudevents.ApplicationJSON, eventData)

	if err := keptnHandler.SendCloudEvent(doneEvent); err != nil {
		keptnHandler.Logger.Error("could not send event: " + err.Error())
	}

	return nil
}
