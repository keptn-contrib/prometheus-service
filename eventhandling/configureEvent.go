package eventhandling

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	cloudevents "github.com/cloudevents/sdk-go"
	"github.com/cloudevents/sdk-go/pkg/cloudevents/client"
	cloudeventshttp "github.com/cloudevents/sdk-go/pkg/cloudevents/transport/http"
	"github.com/cloudevents/sdk-go/pkg/cloudevents/types"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/google/uuid"

	"github.com/keptn-contrib/prometheus-service/utils"

	"github.com/keptn/go-utils/pkg/events"
	"github.com/keptn/go-utils/pkg/models"
	keptnutils "github.com/keptn/go-utils/pkg/utils"

	prometheus_model "github.com/prometheus/common/model"
	prometheusconfig "github.com/prometheus/prometheus/config"
	prometheus_sd_config "github.com/prometheus/prometheus/discovery/config"
	"github.com/prometheus/prometheus/discovery/targetgroup"
)

const configservice = "CONFIGURATION_SERVICE"
const eventbroker = "EVENTBROKER"
const api = "API"

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
}

type alertingAnnotations struct {
	Summary     string `json:"summary" yaml:"summary"`
	Description string `json:"description" yaml:"descriptions"`
}

// GotEvent is the event handler of cloud events
func GotEvent(ctx context.Context, event cloudevents.Event) error {
	var shkeptncontext string
	_ = event.Context.ExtensionAs("shkeptncontext", &shkeptncontext)

	stdLogger := keptnutils.NewLogger(shkeptncontext, event.Context.GetID(), "helm-service")

	var logger keptnutils.LoggerInterface

	connData := &keptnutils.ConnectionData{}
	if err := event.DataAs(connData); err != nil ||
		connData.ChannelInfo.ChannelID == "" || connData.ChannelInfo.Token == "" {
		logger = stdLogger
		logger.Debug("No Websocket connection data available")
	} else {
		apiServiceURL, err := utils.GetServiceEndpoint(api)
		if err != nil {
			logger.Error(err.Error())
			return nil
		}
		ws, _, err := keptnutils.OpenWS(*connData, apiServiceURL)
		defer ws.Close()
		if err != nil {
			stdLogger.Error(fmt.Sprintf("Opening websocket connection failed. %s", err.Error()))
			return nil
		}
		combinedLogger := keptnutils.NewCombinedLogger(stdLogger, ws, shkeptncontext)
		defer combinedLogger.Terminate()
		logger = combinedLogger
	}

	// process event
	if event.Type() == events.ConfigureMonitoringEventType {
		version, err := configurePrometheusAndStoreResources(event, logger)
		if err := logErrAndRespondWithDoneEvent(event, version, err, logger); err != nil {
			return err
		}

		return nil
	}

	const errorMsg = "Received unexpected keptn event that cannot be processed"
	// if err := websocketutil.WriteWSLog(ws, createEventCopy(event, "sh.keptn.events.log"), errorMsg, true, "INFO"); err != nil {
	// 	logger.Error(fmt.Sprintf("Could not write log to websocket. %s", err.Error()))
	// }
	logger.Error(errorMsg)
	return errors.New(errorMsg)
}

// configurePrometheusAndStoreResources
func configurePrometheusAndStoreResources(event cloudevents.Event, logger keptnutils.LoggerInterface) (*models.Version, error) {
	eventData := &events.ConfigureMonitoringEventData{}
	if err := event.DataAs(eventData); err != nil {
		return nil, err
	}

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
	if err := updatePrometheusConfigMap(*eventData, logger); err != nil {
		return nil, err
	}

	// (2.1) delete prometheus pod
	err := deletePrometheusPod()
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func isPrometheusInstalled(logger keptnutils.LoggerInterface) bool {
	logger.Debug("Check if prometheus service in monitoring namespace is available")
	api, err := keptnutils.GetKubeAPI(os.Getenv("env") == "production")
	if err != nil {
		logger.Debug(fmt.Sprintf("Could not initialize kubernetes client %s", err.Error()))
		return false
	}

	_, err = api.Services("monitoring").Get("prometheus-service", metav1.GetOptions{})
	if err != nil {
		logger.Debug(fmt.Sprintf("Prometheus service in monitoring namespace is not available. %s", err.Error()))
		return false
	}

	logger.Debug("Prometheus service in monitoring namespace is available")
	return true
}

func installPrometheus(logger keptnutils.LoggerInterface) error {
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

func installPrometheusAlertManager(logger keptnutils.LoggerInterface) error {
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

func updatePrometheusConfigMap(eventData events.ConfigureMonitoringEventData, logger keptnutils.LoggerInterface) error {
	resourceHandler := keptnutils.NewResourceHandler(getConfigurationServiceURL())
	keptnHandler := keptnutils.NewKeptnHandler(resourceHandler)
	shipyard, err := keptnHandler.GetShipyard(eventData.Project)
	if err != nil {
		return err
	}

	api, err := keptnutils.GetKubeAPI(os.Getenv("env") == "production")
	if err != nil {
		return err
	}

	cmPrometheus, err := api.ConfigMaps("monitoring").Get("prometheus-server-conf", metav1.GetOptions{})
	if err != nil {
		return err
	}
	config, err := prometheusconfig.Load(cmPrometheus.Data["prometheus.yml"])
	if err != nil {
		return err
	}
	fmt.Println(config)

	cmKeptnDomain, err := api.ConfigMaps("keptn").Get("keptn-domain", metav1.GetOptions{})
	if err != nil {
		return err
	}
	gateway := cmKeptnDomain.Data["app_domain"]
	fmt.Println(gateway)

	// check if alerting rules are already availablre
	var alertingRulesConfig alertingRules
	if cmPrometheus.Data["prometheus.rules"] != "" {
		yaml.Unmarshal([]byte(cmPrometheus.Data["prometheus.rules"]), &alertingRulesConfig)
	} else {
		alertingRulesConfig = alertingRules{}
	}
	// update
	for _, stage := range shipyard.Stages {
		var scrapeConfig *prometheusconfig.ScrapeConfig
		scrapeConfigName := eventData.Service + "-" + eventData.Project + "-" + stage.Name
		// (a) if a scrape config with the same name is available, update that one
		scrapeConfig = getScrapeConfig(config, scrapeConfigName)
		// (b) if not, create a new scrape config
		if scrapeConfig == nil {
			scrapeConfig = &prometheusconfig.ScrapeConfig{}
			config.ScrapeConfigs = append(config.ScrapeConfigs, scrapeConfig)
		}
		scrapeConfig.JobName = scrapeConfigName
		scrapeConfig.MetricsPath = "/prometheus"
		scrapeConfig.ServiceDiscoveryConfig = prometheus_sd_config.ServiceDiscoveryConfig{
			StaticConfigs: []*targetgroup.Group{
				{
					Targets: []prometheus_model.LabelSet{
						{prometheus_model.AddressLabel: prometheus_model.LabelValue(eventData.Service + "." + eventData.Project + "-" + stage.Name + "." + gateway + ":80")},
					},
				},
			},
		}

		serviceIndicators, serviceObjectives, err := retrieveMonitoringResources(eventData, stage.Name, logger)
		if err != nil || serviceIndicators == nil || serviceObjectives == nil {
			logger.Info("No SRE files found for stage " + stage.Name + ". No alerting rules created for this stage")
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

		for _, objective := range serviceObjectives.Objectives {

			indicator := getServiceIndicatorForObjective(objective, serviceIndicators)
			if indicator != nil {
				var newAlertingRule *alertingRule
				newAlertingRule = getAlertingRuleOfGroup(alertingGroupConfig, objective.Metric)
				if newAlertingRule == nil {
					newAlertingRule = &alertingRule{
						Alert: objective.Metric,
					}
					alertingGroupConfig.Rules = append(alertingGroupConfig.Rules, newAlertingRule)
				}

				indicatorQueryString := strings.Replace(indicator.Query, "$DURATION_MINUTES", "$DURATION", -1)
				indicatorQueryString = strings.Replace(indicator.Query, "$DURATIONm", "$DURATION", -1)
				indicatorQueryString = strings.Replace(indicatorQueryString, "$DURATION", objective.Timeframe, -1)
				expr := indicatorQueryString + ">" + fmt.Sprintf("%f", objective.Threshold)
				expr = strings.Replace(expr, "$ENVIRONMENT", stage.Name, -1)
				newAlertingRule.Alert = objective.Metric
				newAlertingRule.Expr = expr
				newAlertingRule.For = objective.Timeframe
				newAlertingRule.Labels = &alertingLabel{
					Severity: "webhook",
					PodName:  eventData.Service + "-primary",
				}
				newAlertingRule.Annotations = &alertingAnnotations{
					Summary:     objective.Metric,
					Description: "Pod name {{ $labels.pod_name }}",
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
	_, err = api.ConfigMaps("monitoring").Update(cmPrometheus)
	if err != nil {
		return err
	}
	return nil
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
	if os.Getenv("env") == "production" {
		return "configuration-service.keptn.svc.cluster.local:8080"
	}
	return "localhost:6060"
}

func getServiceIndicatorForObjective(objective *models.ServiceObjective, indicators *models.ServiceIndicators) *models.ServiceIndicator {
	for _, indicator := range indicators.Indicators {
		if indicator.Metric == objective.Metric && strings.ToLower(indicator.Source) == "prometheus" {
			return indicator
		}
	}
	return nil
}

func deletePrometheusPod() error {

	if err := keptnutils.RestartPodsWithSelector(os.Getenv("env") == "production", "monitoring", "app=prometheus-server"); err != nil {
		return err
	}
	return nil
}

func retrieveMonitoringResources(eventData events.ConfigureMonitoringEventData, stage string, logger keptnutils.LoggerInterface) (*models.ServiceIndicators, *models.ServiceObjectives, error) {
	resourceHandler := keptnutils.NewResourceHandler(getConfigurationServiceURL())

	resource, err := resourceHandler.GetServiceResource(eventData.Project, stage, eventData.Service, "service-indicators.yaml")
	if err != nil || resource.ResourceContent == "" {
		return nil, nil, errors.New("No Service indicators file available for service " + eventData.Service + " in stage " + stage)
	}
	var serviceIndicators models.ServiceIndicators

	err = yaml.Unmarshal([]byte(resource.ResourceContent), &serviceIndicators)
	if err != nil {
		return nil, nil, errors.New("Invalid Service indicators file format")
	}

	resource, err = resourceHandler.GetServiceResource(eventData.Project, stage, eventData.Service, "service-objectives.yaml")
	if err != nil || resource.ResourceContent == "" {
		return nil, nil, errors.New("No Service objectives file available for service " + eventData.Service + " in stage " + stage)
	}
	var serviceObjectives models.ServiceObjectives

	err = yaml.Unmarshal([]byte(resource.ResourceContent), &serviceObjectives)
	if err != nil {
		return nil, nil, errors.New("Invalid Service objectives file format")
	}

	return &serviceIndicators, &serviceObjectives, nil

}

// logErrAndRespondWithDoneEvent sends a keptn done event to the keptn eventbroker
func logErrAndRespondWithDoneEvent(event cloudevents.Event, version *models.Version, err error, logger keptnutils.LoggerInterface) error {
	var result = "success"
	//var webSocketMessage = "Prometheus successfully configured"
	var eventMessage = "Prometheus successfully configured and rule created"

	if err != nil { // error
		result = "error"
		eventMessage = fmt.Sprintf("%s.", err.Error())
		//webSocketMessage = eventMessage
		logger.Error(eventMessage)
	} else { // success
		logger.Info(eventMessage)
	}

	// if err := websocketutil.WriteWSLog(ws, createEventCopy(event, "sh.keptn.events.log"), webSocketMessage, true, "INFO"); err != nil {
	// 	logger.Error(fmt.Sprintf("Could not write log to websocket. %s", err.Error()))
	// }
	if err := sendDoneEvent(event, result, eventMessage, version); err != nil {
		logger.Error(fmt.Sprintf("No sh.keptn.event.done event sent. %s", err.Error()))
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
	contentType := "application/json"

	event := cloudevents.Event{
		Context: cloudevents.EventContextV02{
			ID:          uuid.New().String(),
			Time:        &types.Timestamp{Time: time.Now()},
			Type:        eventType,
			Source:      types.URLRef{URL: *source},
			ContentType: &contentType,
			Extensions: map[string]interface{}{
				"shkeptncontext": shkeptncontext,
				"shkeptnphaseid": shkeptnphaseid,
				"shkeptnphase":   shkeptnphase,
				"shkeptnstepid":  shkeptnstepid,
				"shkeptnstep":    shkeptnstep,
			},
		}.AsV02(),
	}

	return event
}

// sendDoneEvent prepares a keptn done event and sends it to the eventbroker
func sendDoneEvent(receivedEvent cloudevents.Event, result string, message string, version *models.Version) error {

	doneEvent := createEventCopy(receivedEvent, "sh.keptn.events.done")

	eventData := doneEventData{
		Result:  result,
		Message: message,
	}

	if version != nil {
		eventData.Version = version.Version
	}

	doneEvent.Data = eventData

	endPoint, err := utils.GetServiceEndpoint(eventbroker)
	if err != nil {
		return errors.New("Failed to retrieve endpoint of eventbroker. %s" + err.Error())
	}

	if endPoint.Host == "" {
		return errors.New("Host of eventbroker not set")
	}

	transport, err := cloudeventshttp.New(
		cloudeventshttp.WithTarget(endPoint.String()),
		cloudeventshttp.WithEncoding(cloudeventshttp.StructuredV02),
	)
	if err != nil {
		return errors.New("Failed to create transport: " + err.Error())
	}

	client, err := client.New(transport)
	if err != nil {
		return errors.New("Failed to create HTTP client: " + err.Error())
	}

	if _, err := client.Send(context.Background(), doneEvent); err != nil {
		return errors.New("Failed to send cloudevent sh.keptn.events.done: " + err.Error())
	}

	return nil
}
