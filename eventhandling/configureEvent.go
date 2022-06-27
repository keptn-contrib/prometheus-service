package eventhandling

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	configutils "github.com/keptn/go-utils/pkg/api/utils"
	keptnevents "github.com/keptn/go-utils/pkg/lib"
	"github.com/keptn/go-utils/pkg/lib/keptn"
	keptnv2 "github.com/keptn/go-utils/pkg/lib/v0_2_0"

	"github.com/gobwas/glob"
	"github.com/keptn-contrib/prometheus-service/utils"
	"github.com/keptn-contrib/prometheus-service/utils/prometheus"
	prometheus_model "github.com/prometheus/common/model"
)

const metricsScrapePathEnvName = "METRICS_SCRAPE_PATH"

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

// HandleEvent processes an event
func (eh ConfigureMonitoringEventHandler) HandleEvent() error {
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
		return eh.handleError(err.Error())
	}

	if err = eh.sendConfigureMonitoringFinishedEvent(keptnv2.StatusSucceeded, keptnv2.ResultPass, "Prometheus successfully configured and rule created"); err != nil {
		eh.logger.Error(err.Error())
	}
	return nil
}

// configurePrometheusAndStoreResources
func (eh ConfigureMonitoringEventHandler) configurePrometheusAndStoreResources(eventData *keptnevents.ConfigureMonitoringEventData) error {
	// (1) check if prometheus is installed
	if eh.isPrometheusInstalled() {
		if utils.EnvVarOrDefault("CREATE_TARGETS", "true") == "true" {
			eh.logger.Debug("Configure prometheus monitoring with keptn")
			if err := eh.updatePrometheusConfigMap(*eventData); err != nil {
				return err
			}
		}

		if utils.EnvVarOrDefault("CREATE_ALERTS", "true") == "true" {
			eh.logger.Debug("Configure prometheus alert manager with keptn")
			err := eh.configurePrometheusAlertManager()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (eh ConfigureMonitoringEventHandler) isPrometheusInstalled() bool {
	eh.logger.Debug("Check if prometheus service in " + env.PrometheusNamespace + " namespace is available")
	svcList, err := getPrometheusServiceFromK8s()
	if err != nil {
		eh.logger.Errorf("Error locating prometheus service in k8s: %v", err)
		return false
	}

	if len(svcList.Items) > 0 {
		eh.logger.Debug("Prometheus service in " + env.PrometheusNamespace + " namespace is available")
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

func (eh ConfigureMonitoringEventHandler) configurePrometheusAlertManager() error {
	eh.logger.Info("Configuring Prometheus AlertManager...")
	prometheusHelper, err := prometheus.NewPrometheusHelper()

	eh.logger.Info("Updating Prometheus AlertManager configmap...")
	err = prometheusHelper.UpdateAMConfigMap(env.AlertManagerConfigMap, env.AlertManagerConfigFileName, env.AlertManagerNamespace)
	if err != nil {
		return err
	}

	eh.logger.Info("Prometheus AlertManager configuration successfully")

	return nil
}

// updatePrometheusConfigMap updates the prometheus configmap with scrape configs and alerting rules
func (eh ConfigureMonitoringEventHandler) updatePrometheusConfigMap(eventData keptnevents.ConfigureMonitoringEventData) error {
	shipyard, err := eh.keptnHandler.GetShipyard()
	if err != nil {
		return err
	}

	api, err := utils.GetKubeClient()
	if err != nil {
		return err
	}

	cmPrometheus, err := api.CoreV1().ConfigMaps(env.PrometheusNamespace).Get(context.TODO(), env.PrometheusConfigMap, metav1.GetOptions{})
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
		eh.logger.Error("Error while converting SCRAPE_INTERVAL value. Using default value instead!")
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

		alertingRulesConfig, err = eh.createPrometheusAlertsIfSLOsAndRemediationDefined(eventData, stage,
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
	_, err = api.CoreV1().ConfigMaps(env.PrometheusNamespace).Update(context.TODO(), cmPrometheus, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (eh ConfigureMonitoringEventHandler) createPrometheusAlertsIfSLOsAndRemediationDefined(
	eventData keptnevents.ConfigureMonitoringEventData, stage keptnv2.Stage, alertingRulesConfig alertingRules,
) (alertingRules, error) {
	// fetch SLOs for the given service and stage
	slos, err := retrieveSLOs(eventData, stage.Name, eh.logger)
	if err != nil || slos == nil {
		eh.logger.Info("No SLO file found for stage " + stage.Name + ". No alerting rules created for this stage")
		return alertingRulesConfig, nil
	}

	const remediationFileDefaultName = "remediation.yaml"

	resourceScope := configutils.NewResourceScope()
	resourceScope.Project(eventData.Project)
	resourceScope.Service(eventData.Service)
	resourceScope.Stage(stage.Name)
	resourceScope.Resource(remediationFileDefaultName)

	_, err = eh.keptnHandler.ResourceHandler.GetResource(*resourceScope)

	if errors.Is(err, configutils.ResourceNotFoundError) {
		eh.logger.Infof("No remediation defined for project %s stage %s, skipping setup of prometheus alerts",
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
	prometheusHandler := prometheus.NewPrometheusHandler(
		"",
		&keptnv2.EventData{
			Project: eventData.Project,
			Service: eventData.Service,
			Stage:   stage.Name,
		},
		"primary", // only create alerts for primary deployments
		nil,
		nil,
	)

	// get SLI queries
	projectCustomQueries, err := getCustomQueries(eh.keptnHandler, eventData.Project, stage.Name, eventData.Service)
	if err != nil {
		log.Println("Failed to get custom queries for project " + eventData.Project)
		log.Println(err.Error())
		return alertingRulesConfig, err
	}

	if projectCustomQueries != nil {
		prometheusHandler.CustomQueries = projectCustomQueries
	}

	eh.logger.Info("Going over SLO.objectives")

	for _, objective := range slos.Objectives {
		eh.logger.Info("SLO:" + objective.DisplayName + ", " + objective.SLI)
		// Get Prometheus Metric Expression
		end := time.Now()
		start := end.Add(-180 * time.Second)

		expr, err := prometheusHandler.GetMetricQuery(objective.SLI, start, end)
		if err != nil || expr == "" {
			eh.logger.Error("No query defined for SLI " + objective.SLI + " in project " + eventData.Project)
			continue
		}
		eh.logger.Info("expr=" + expr)

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

func getConfigurationServiceURL() string {
	return env.ConfigurationServiceURL
}

func retrieveSLOs(eventData keptnevents.ConfigureMonitoringEventData, stage string, logger keptn.LoggerInterface) (*keptnevents.ServiceLevelObjectives, error) {
	resourceHandler := configutils.NewResourceHandler(getConfigurationServiceURL())

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

func (eh ConfigureMonitoringEventHandler) sendConfigureMonitoringFinishedEvent(status keptnv2.StatusType, result keptnv2.ResultType, msg string) error {
	_, err := eh.keptnHandler.SendTaskFinishedEvent(&keptnv2.EventData{
		Status:  status,
		Result:  result,
		Message: msg,
	}, utils.ServiceName)

	if err != nil {
		return fmt.Errorf("could not send %s event: %s", keptnv2.GetFinishedEventType(keptnv2.ConfigureMonitoringTaskName), err.Error())
	}

	return nil
}

func (eh ConfigureMonitoringEventHandler) handleError(msg string) error {
	//logger.Error(msg)
	if err := eh.sendConfigureMonitoringFinishedEvent(keptnv2.StatusErrored, keptnv2.ResultFailed, msg); err != nil {
		// an additional error occurred when trying to send configure monitoring finished back to Keptn
		eh.logger.Error(err.Error())
	}
	return errors.New(msg)
}
