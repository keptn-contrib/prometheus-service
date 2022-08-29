package eventhandling

import (
	"context"
	"errors"
	"fmt"
	"github.com/kelseyhightower/envconfig"
	"github.com/keptn/go-utils/pkg/sdk"
	"log"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	configutils "github.com/keptn/go-utils/pkg/api/utils"
	keptnevents "github.com/keptn/go-utils/pkg/lib"
	keptnv2 "github.com/keptn/go-utils/pkg/lib/v0_2_0"

	"github.com/gobwas/glob"
	"github.com/keptn-contrib/prometheus-service/utils"
	"github.com/keptn-contrib/prometheus-service/utils/prometheus"
	api "github.com/keptn/go-utils/pkg/api/utils"
	prometheus_model "github.com/prometheus/common/model"
)

const metricsScrapePathEnvName = "METRICS_SCRAPE_PATH"

// ConfigureMonitoringEventHandler is responsible for processing configure monitoring events
type ConfigureMonitoringEventHandler struct {
}

// NewConfigureMonitoringEventHandler creates a new ConfigureMonitoringEventHandler
func NewConfigureMonitoringEventHandler() *ConfigureMonitoringEventHandler {
	return &ConfigureMonitoringEventHandler{}
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
	Severity   string `json:"severity" yaml:"severity"`
	PodName    string `json:"pod_name,omitempty" yaml:"pod_name"`
	Service    string `json:"service,omitempty" yaml:"service"`
	Stage      string `json:"stage,omitempty" yaml:"stage"`
	Project    string `json:"project,omitempty" yaml:"project"`
	Deployment string `json:"deployment,omitempty" yaml:"deployment"`
}

type alertingAnnotations struct {
	Summary     string `json:"summary" yaml:"summary"`
	Description string `json:"description" yaml:"descriptions"`
}

// Execute processes an event
func (eh ConfigureMonitoringEventHandler) Execute(k sdk.IKeptn, event sdk.KeptnEvent) (interface{}, *sdk.Error) {
	k.Logger().Infof("Handling configure monitoring event from %s with id: %s and context: %s", *event.Source, event.ID, event.Shkeptncontext)

	if err := envconfig.Process("", &env); err != nil {
		k.Logger().Error("Failed to process env var: " + err.Error())
	}

	eventData := &keptnevents.ConfigureMonitoringEventData{}
	if err := keptnv2.Decode(event.Data, eventData); err != nil {
		return nil, &sdk.Error{Err: err, StatusType: keptnv2.StatusErrored, ResultType: keptnv2.ResultFailed, Message: "failed to decode get-sli.triggered event: " + err.Error()}
	}

	if err := eh.sendConfigureMonitoringStartedEvent(k, event); err != nil {
		k.Logger().Infof("Error while sending configure-monitoring.started event: %s", err.Message)
		return nil, err
	}

	err := eh.configurePrometheusAndStoreResources(k, eventData, os.Getenv("K8S_NAMESPACE"))
	if err != nil {
		k.Logger().Error(err.Error())
		return nil, &sdk.Error{Err: err, StatusType: keptnv2.StatusErrored, ResultType: keptnv2.ResultFailed, Message: "configure prometheus failed with error: " + err.Error()}
	}

	finishedEventData := eh.getConfigureMonitoringFinishedEvent(keptnv2.StatusSucceeded, keptnv2.ResultPass, *eventData, "Prometheus successfully configured and rule created")
	k.Logger().Infof("Sending configure-monitoring.finished event with context: %s", event.Shkeptncontext)
	if err := eh.sendConfigureMonitoringFinishedEvent(k, event, finishedEventData); err != nil {
		k.Logger().Infof("Error while sending configure-monitoring.finished event: %s", err.Message)
		return nil, err
	}

	return finishedEventData, nil
}

// configurePrometheusAndStoreResources
func (eh ConfigureMonitoringEventHandler) configurePrometheusAndStoreResources(k sdk.IKeptn, eventData *keptnevents.ConfigureMonitoringEventData, k8sNamespace string) error {
	// (1) check if prometheus is installed
	if eh.isPrometheusInstalled(k) {
		if utils.EnvVarOrDefault("CREATE_TARGETS", "true") == "true" {
			k.Logger().Debug("Configure prometheus monitoring with keptn")
			if err := eh.updatePrometheusConfigMap(k, *eventData); err != nil {
				return err
			}
		}

		if utils.EnvVarOrDefault("CREATE_ALERTS", "true") == "true" {
			k.Logger().Debug("Configure prometheus alert manager with keptn")
			err := eh.configurePrometheusAlertManager(k, k8sNamespace)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (eh ConfigureMonitoringEventHandler) isPrometheusInstalled(k sdk.IKeptn) bool {
	k.Logger().Debug("Check if prometheus service in " + env.PrometheusNamespace + " namespace is available")
	svcList, err := getPrometheusServiceFromK8s()
	if err != nil {
		k.Logger().Errorf("Error locating prometheus service in k8s: %v", err)
		return false
	}

	if len(svcList.Items) > 0 {
		k.Logger().Debug("Prometheus service in " + env.PrometheusNamespace + " namespace is available")
		return true
	}

	return false
}

func getPrometheusServiceFromK8s() (*v1.ServiceList, error) {
	svcList, err := utils.ListK8sServicesByLabel(env.PrometheusLabels, env.PrometheusNamespace)
	if err != nil {
		return nil, fmt.Errorf("prometheus service not found: %w", err)
	}
	return svcList, err
}

func getPrometheusAlertManagerServiceFromK8s() (*v1.ServiceList, error) {
	svcList, err := utils.ListK8sServicesByLabel(env.AlertManagerLabels, env.AlertManagerNamespace)
	if err != nil {
		return nil, fmt.Errorf("prometheus alert manager service not found: %w", err)
	}
	return svcList, err
}

func (eh ConfigureMonitoringEventHandler) configurePrometheusAlertManager(k sdk.IKeptn, namespace string) error {
	k.Logger().Info("Configuring Prometheus AlertManager...")
	prometheusHelper, err := prometheus.NewPrometheusHelper(namespace)

	k.Logger().Info("Updating Prometheus AlertManager configmap...")
	err = prometheusHelper.UpdateAMConfigMap(env.AlertManagerConfigMap, env.AlertManagerConfigFileName, env.AlertManagerNamespace)
	if err != nil {
		return err
	}

	k.Logger().Info("Prometheus AlertManager configuration successfully")

	return nil
}

// updatePrometheusConfigMap updates the prometheus configmap with scrape configs and alerting rules
func (eh ConfigureMonitoringEventHandler) updatePrometheusConfigMap(k sdk.IKeptn, eventData keptnevents.ConfigureMonitoringEventData) error {
	scope := api.NewResourceScope()
	scope.Project(eventData.Project)
	scope.Resource("shipyard.yaml")

	shipyard, err := GetShipyard(k.GetResourceHandler(), *scope)
	if err != nil {
		return err
	}

	kubeAPI, err := utils.GetKubeClient()
	if err != nil {
		return err
	}

	cmPrometheus, err := kubeAPI.CoreV1().ConfigMaps(env.PrometheusNamespace).Get(context.TODO(), env.PrometheusConfigMap, metav1.GetOptions{})
	if err != nil {
		// Print better error message when role binding is missing
		g := glob.MustCompile("configmaps * is forbidden: User * cannot get resource * in API group * in the namespace *")
		if g.Match(err.Error()) {
			return errors.New("not enough permissions to access configmap. Check if the role binding is correct")
		}
		return err
	}
	config, err := prometheus.LoadYamlConfiguration(cmPrometheus.Data[env.PrometheusConfigFileName])
	if err != nil {
		return err
	}

	scrapeIntervalString := utils.EnvVarOrDefault("SCRAPE_INTERVAL", "5s")
	scrapeInterval, err := time.ParseDuration(scrapeIntervalString)

	if err != nil {
		k.Logger().Error("Error while converting SCRAPE_INTERVAL value. Using default value instead!")
		scrapeInterval = 5 * time.Second
	}

	// check if alerting rules are already available
	var alertingRulesConfig alertingRules
	if cmPrometheus.Data["prometheus.rules"] != "" {
		// take existing alerting rule
		err := yaml.Unmarshal([]byte(cmPrometheus.Data["prometheus.rules"]), &alertingRulesConfig)
		if err != nil {
			return fmt.Errorf("unable to parse altering rules configuration: %w", err)
		}
	} else {
		// create new empty alerting rule
		alertingRulesConfig = alertingRules{}
	}
	// update: Create scrape job and alerting rules for each stage of the shipyard file
	for _, stage := range shipyard.Spec.Stages {
		var scrapeConfig *prometheus.ScrapeConfig
		// (a) if a scrape config with the same name is available, update that one

		// <service>-primary.<project>-<stage>
		createScrapeJobConfig(scrapeConfig, config, eventData.Project, stage.Name, eventData.Service, false, true, scrapeInterval)
		// <service>-canary.<project>-<stage>
		createScrapeJobConfig(scrapeConfig, config, eventData.Project, stage.Name, eventData.Service, true, false, scrapeInterval)
		// <service>.<project>-<stage>
		createScrapeJobConfig(scrapeConfig, config, eventData.Project, stage.Name, eventData.Service, false, false, scrapeInterval)

		alertingRulesConfig, err = eh.createPrometheusAlertsIfSLOsAndRemediationDefined(k, eventData, stage,
			alertingRulesConfig)

		if err != nil {
			return fmt.Errorf("error configuring prometheus alerts: %w", err)
		}
	}
	alertingRulesYAMLString, err := yaml.Marshal(alertingRulesConfig)
	if err != nil {
		return err
	}

	updatedConfigYAMLString, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	// apply
	cmPrometheus.Data["alerting_rules.yml"] = string(alertingRulesYAMLString)
	cmPrometheus.Data[env.PrometheusConfigFileName] = string(updatedConfigYAMLString)
	_, err = kubeAPI.CoreV1().ConfigMaps(env.PrometheusNamespace).Update(context.TODO(), cmPrometheus, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (eh ConfigureMonitoringEventHandler) createPrometheusAlertsIfSLOsAndRemediationDefined(
	k sdk.IKeptn, eventData keptnevents.ConfigureMonitoringEventData, stage keptnv2.Stage, alertingRulesConfig alertingRules,
) (alertingRules, error) {
	// fetch SLOs for the given service and stage
	slos, err := retrieveSLOs(k.GetResourceHandler(), eventData, stage.Name)
	if err != nil || slos == nil {
		k.Logger().Info("No SLO file found for stage " + stage.Name + ". No alerting rules created for this stage")
		return alertingRulesConfig, nil
	}

	const remediationFileDefaultName = "remediation.yaml"

	resourceScope := configutils.NewResourceScope()
	resourceScope.Project(eventData.Project)
	resourceScope.Service(eventData.Service)
	resourceScope.Stage(stage.Name)
	resourceScope.Resource(remediationFileDefaultName)

	_, err = k.GetResourceHandler().GetResource(*resourceScope)

	if errors.Is(err, configutils.ResourceNotFoundError) {
		k.Logger().Infof("No remediation defined for project %s stage %s, skipping setup of prometheus alerts",
			eventData.Project, stage.Name)
		return alertingRulesConfig, nil
	}

	if err != nil {
		return alertingRulesConfig,
			fmt.Errorf("error retrieving remediation definition %s for project %s and stage %s: %w",
				remediationFileDefaultName, eventData.Project, stage.Name, err)
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

	// create a new prometheus handler in order to query SLI expressions
	const deploymentType = "primary" // only create alerts for primary deployments
	prometheusHandler := prometheus.NewPrometheusHandler(
		"",
		&keptnv2.EventData{
			Project: eventData.Project,
			Service: eventData.Service,
			Stage:   stage.Name,
		},
		deploymentType,
		nil,
		nil,
	)

	// get SLI queries
	projectCustomQueries, err := getCustomQueries(k.GetResourceHandler(), eventData.Project, stage.Name, eventData.Service)
	if err != nil {
		log.Println("Failed to get custom queries for project " + eventData.Project)
		log.Println(err.Error())
		return alertingRulesConfig, err
	}

	if projectCustomQueries != nil {
		prometheusHandler.CustomQueries = projectCustomQueries
	}

	k.Logger().Info("Going over SLO.objectives")

	for _, objective := range slos.Objectives {
		k.Logger().Info("SLO:" + objective.DisplayName + ", " + objective.SLI)
		// Get Prometheus Metric Expression
		end := time.Now()
		start := end.Add(-180 * time.Second)

		expr, err := prometheusHandler.GetMetricQuery(objective.SLI, start, end)
		if err != nil || expr == "" {
			k.Logger().Error("No query defined for SLI " + objective.SLI + " in project " + eventData.Project)
			continue
		}
		k.Logger().Info("expr=" + expr)

		if objective.Pass != nil {
			for _, criteriaGroup := range objective.Pass {
				for _, criteria := range criteriaGroup.Criteria {
					if strings.Contains(criteria, "+") || strings.Contains(criteria, "-") || strings.Contains(
						criteria, "%",
					) || (!strings.Contains(criteria, "<") && !strings.Contains(criteria, ">")) {
						continue
					}
					criteriaString := strings.Replace(criteria, "=", "", -1)
					if strings.Contains(criteriaString, "<") {
						criteriaString = strings.Replace(criteriaString, "<", ">", -1)
					} else {
						criteriaString = strings.Replace(criteriaString, ">", "<", -1)
					}

					// sanitize criteria string: remove whitespaces
					criteriaString = strings.Replace(criteriaString, " ", "", -1)

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
						Severity:   "webhook",
						PodName:    fmt.Sprintf("%s-%s", eventData.Service, deploymentType),
						Service:    eventData.Service,
						Project:    eventData.Project,
						Stage:      stage.Name,
						Deployment: deploymentType,
					}
					newAlertingRule.Annotations = &alertingAnnotations{
						Summary:     ruleName,
						Description: "Pod name {{ $labels.pod_name }}",
					}
				}
			}
		}
	}

	return alertingRulesConfig, nil
}

func createScrapeJobConfig(scrapeConfig *prometheus.ScrapeConfig, config *prometheus.Config, project string, stage string, service string, isCanary bool, isPrimary bool, scrapeInterval time.Duration) {
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
		scrapeConfig = &prometheus.ScrapeConfig{}
		config.ScrapeConfigs = append(config.ScrapeConfigs, scrapeConfig)
	}

	// define scrape job name
	scrapeConfig.JobName = scrapeConfigName
	// set scrape interval
	scrapeConfig.ScrapeInterval = prometheus_model.Duration(scrapeInterval)
	scrapeConfig.ScrapeTimeout = prometheus_model.Duration(3 * time.Second)
	// configure metrics path (default: /metrics)
	scrapeConfig.MetricsPath = utils.EnvVarOrDefault(metricsScrapePathEnvName, "/metrics")
	scrapeConfig.StaticConfigs = prometheus.Configs{
		prometheus.StaticConfigLike{
			Targets: []string{
				scrapeEndpoint,
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

func getScrapeConfig(config *prometheus.Config, name string) *prometheus.ScrapeConfig {
	for _, scrapeConfig := range config.ScrapeConfigs {
		if scrapeConfig.JobName == name {
			return scrapeConfig
		}
	}
	return nil
}

func retrieveSLOs(resourceHandler sdk.ResourceHandler, eventData keptnevents.ConfigureMonitoringEventData, stage string) (*keptnevents.ServiceLevelObjectives, error) {
	resourceScope := configutils.NewResourceScope()
	resourceScope.Project(eventData.Project)
	resourceScope.Service(eventData.Service)
	resourceScope.Stage(stage)
	resourceScope.Resource("slo.yaml")

	resource, err := resourceHandler.GetResource(*resourceScope)
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

func (eh ConfigureMonitoringEventHandler) getConfigureMonitoringFinishedEvent(status keptnv2.StatusType, result keptnv2.ResultType, configureMonitoringTriggeredEven keptnevents.ConfigureMonitoringEventData, msg string) keptnv2.ConfigureMonitoringFinishedEventData {

	return keptnv2.ConfigureMonitoringFinishedEventData{
		EventData: keptnv2.EventData{
			Project: configureMonitoringTriggeredEven.Project,
			Service: configureMonitoringTriggeredEven.Service,
			Status:  status,
			Result:  result,
			Message: msg,
		},
	}
}

func (eh ConfigureMonitoringEventHandler) sendConfigureMonitoringStartedEvent(k sdk.IKeptn, event sdk.KeptnEvent) *sdk.Error {
	eventType := keptnv2.GetTriggeredEventType(keptnv2.ConfigureMonitoringTaskName)
	event.Type = &eventType

	if err := k.SendStartedEvent(event); err != nil {
		return &sdk.Error{Err: err, StatusType: keptnv2.StatusErrored, ResultType: keptnv2.ResultFailed, Message: "Error sending configure-monitoring.started: " + err.Error()}
	}
	return nil
}

func (eh ConfigureMonitoringEventHandler) sendConfigureMonitoringFinishedEvent(k sdk.IKeptn, event sdk.KeptnEvent, eventData keptnv2.ConfigureMonitoringFinishedEventData) *sdk.Error {
	eventType := keptnv2.GetTriggeredEventType(keptnv2.ConfigureMonitoringTaskName)
	event.Type = &eventType

	if err := k.SendFinishedEvent(event, eventData); err != nil {
		return &sdk.Error{Err: err, StatusType: keptnv2.StatusErrored, ResultType: keptnv2.ResultFailed, Message: "Error sending configure-monitoring.started: " + err.Error()}
	}
	return nil
}

// GetShipyard returns the shipyard definition of a project
func GetShipyard(resourceHandler sdk.ResourceHandler, scope api.ResourceScope) (*keptnv2.Shipyard, error) {
	shipyardResource, err := resourceHandler.GetResource(scope)
	if err != nil {
		return nil, err
	}

	shipyard := keptnv2.Shipyard{}
	err = yaml.Unmarshal([]byte(shipyardResource.ResourceContent), &shipyard)
	if err != nil {
		return nil, err
	}
	return &shipyard, nil
}
