package eventhandling

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"

	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cloudevents "github.com/cloudevents/sdk-go"
	"github.com/cloudevents/sdk-go/pkg/cloudevents/client"
	cloudeventshttp "github.com/cloudevents/sdk-go/pkg/cloudevents/transport/http"
	"github.com/cloudevents/sdk-go/pkg/cloudevents/types"

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
	Groups []alertingGroup `json:"groups" yaml:"groups"`
}

type alertingGroup struct {
	Name  string         `json:"name" yaml:"name"`
	Rules []alertingRule `json:"rules" yaml:"rules"`
}

type alertingRule struct {
	Alert       string              `json:"alert" yaml:"alert"`
	Expr        string              `json:"expr" yaml:"expr"`
	For         string              `json:"for" yaml:"for"`
	Labels      alertingLabel       `json:"labels" yaml:"labels"`
	Annotations alertingAnnotations `json:"annotations" yaml:"annotations"`
}

type alertingLabel struct {
	Severity string `json:"severity" yaml:"severity"`
}

type alertingAnnotations struct {
	Summary     string `json:"summary" yaml:"summary"`
	Description string `json:"description" yaml:"descriptions"`
}

type options []string

// GotEvent is the event handler of cloud events
func GotEvent(ctx context.Context, event cloudevents.Event) error {
	var shkeptncontext string
	event.Context.ExtensionAs("shkeptncontext", &shkeptncontext)

	logger := keptnutils.NewLogger(shkeptncontext, event.Context.GetID(), "prometheus-service")

	// open websocket connection to api component
	// endPoint, err := utils.GetServiceEndpoint(api)
	// if err != nil {
	// 	return err
	// }

	// if endPoint.Host == "" {
	// 	const errorMsg = "Host of api not set"
	// 	logger.Error(errorMsg)
	// 	return errors.New(errorMsg)
	// }

	// connData := &websockethelper.ConnectionData{}
	// if err := event.DataAs(connData); err != nil {
	// 	logger.Error(fmt.Sprintf("Data of the event is incompatible. %s", err.Error()))
	// 	return err
	// }

	// ws, _, err := websocketutil.OpenWS(*connData, endPoint)
	// if err != nil {
	// 	logger.Error(fmt.Sprintf("Opening websocket connection failed. %s", err.Error()))
	// 	return err
	// }
	// defer ws.Close()

	// process event
	if event.Type() == events.ConfigureMonitoringEventType {
		version, err := configurePrometheusAndStoreResources(event, *logger)
		if err := logErrAndRespondWithDoneEvent(event, version, err, *logger); err != nil {
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
func configurePrometheusAndStoreResources(event cloudevents.Event, logger keptnutils.Logger) (*models.Version, error) {
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

	// (2) update config map with alert rule
	if err := updatePrometheusConfigMap(*eventData, logger); err != nil {
		return nil, err
	}

	// (2.1) delete prometheus pod
	err := deletePrometheusPod()
	if err != nil {
		return nil, err
	}

	// (3) store resources
	return storeMonitoringResources(*eventData, logger)
	return nil, nil
}

func isPrometheusInstalled(logger keptnutils.Logger) bool {
	logger.Debug("Check if prometheus service in monitoring namespace is available")

	o := options{"get", "svc", "prometheus-service", "-n", "monitoring"}
	_, err := keptnutils.ExecuteCommand("kubectl", o)
	if err != nil {
		logger.Debug(fmt.Sprintf("Prometheus service in monitoring namespace is not available. %s", err.Error()))
		return false
	}

	logger.Debug("Prometheus service in monitoring namespace is available")
	return true
}

func installPrometheus(logger keptnutils.Logger) error {
	//namespace.yaml
	logger.Debug("Apply namespace for prometheus monitoring")
	o := options{"apply", "-f", "/manifests/namespace.yaml"}
	_, err := keptnutils.ExecuteCommand("kubectl", o)
	if err != nil {
		return err
	}

	//config-map.yaml
	logger.Debug("Apply configmap for prometheus monitoring")
	o = options{"apply", "-f", "/manifests/config-map.yaml"}
	_, err = keptnutils.ExecuteCommand("kubectl", o)
	if err != nil {
		return err
	}

	//cluster-role.yaml
	logger.Debug("Apply clusterrole for prometheus monitoring")
	o = options{"apply", "-f", "/manifests/cluster-role.yaml"}
	_, err = keptnutils.ExecuteCommand("kubectl", o)
	if err != nil {
		return err
	}

	//prometheus.yaml
	logger.Debug("Apply service and deployment for prometheus monitoring")
	o = options{"apply", "-f", "/manifests/prometheus.yaml"}
	_, err = keptnutils.ExecuteCommand("kubectl", o)
	if err != nil {
		return err
	}

	return nil
}

func installPrometheusAlertManager(logger keptnutils.Logger) error {
	//alertmanager-configmap.yaml
	logger.Debug("Apply configmap for prometheus alert manager")
	o := options{"apply", "-f", "/manifests/alertmanager-configmap.yaml"}
	_, err := keptnutils.ExecuteCommand("kubectl", o)
	if err != nil {
		return err
	}

	//alertmanager-template.yaml
	logger.Debug("Apply configmap template for prometheus alert manager")
	o = options{"apply", "-f", "/manifests/alertmanager-template.yaml"}
	_, err = keptnutils.ExecuteCommand("kubectl", o)
	if err != nil {
		return err
	}

	//alertmanager-deployment.yaml
	logger.Debug("Apply deployment for prometheus alert manager")
	o = options{"apply", "-f", "/manifests/alertmanager-deployment.yaml"}
	_, err = keptnutils.ExecuteCommand("kubectl", o)
	if err != nil {
		return err
	}

	//alertmanager-svc.yaml
	logger.Debug("Apply service for prometheus alert manager")
	o = options{"apply", "-f", "/manifests/alertmanager-deployment.yaml"}
	_, err = keptnutils.ExecuteCommand("kubectl", o)
	if err != nil {
		return err
	}

	return nil
}

func updatePrometheusConfigMap(eventData events.ConfigureMonitoringEventData, logger keptnutils.Logger) error {
	api, err := keptnutils.GetKubeAPI(os.Getenv("env") == "production")
	if err != nil {
		return err
	}

	cmPrometheus, err := api.ConfigMaps("monitoring").Get("prometheus-server-conf", metav1.GetOptions{})
	if err != nil {
		return err
	}
	config, err := prometheusconfig.Load(cmPrometheus.Data["prometheus.yml"])
	fmt.Print(config)

	cmKeptnDomain, err := api.ConfigMaps("keptn").Get("keptn-domain", metav1.GetOptions{})
	if err != nil {
		return err
	}
	gateway := cmKeptnDomain.Data["app_domain"]
	fmt.Print(gateway)

	configEndpoint, err := utils.GetServiceEndpoint(configservice)
	resourceHandler := keptnutils.NewResourceHandler(configEndpoint.Host)
	keptnHandler := keptnutils.NewKeptnHandler(resourceHandler)
	shipyard, err := keptnHandler.GetShipyard(eventData.Project)
	if err != nil {
		return err
	}
	var ars alertingRules
	if cmPrometheus.Data["prometheus.rules"] != "" {
		yaml.Unmarshal([]byte(cmPrometheus.Data["prometheus.rules"]), &ars)
	} else {
		ars = alertingRules{
			Groups: []alertingGroup{},
		}
	}
	// update
	for _, stage := range shipyard.Stages {

		// create scrape config
		scrapeConfig := &prometheusconfig.ScrapeConfig{
			JobName:     eventData.Service + "-" + eventData.Project + "-" + stage.Name,
			MetricsPath: "/prometheus",
			ServiceDiscoveryConfig: prometheus_sd_config.ServiceDiscoveryConfig{
				StaticConfigs: []*targetgroup.Group{
					{
						Targets: []prometheus_model.LabelSet{
							{prometheus_model.AddressLabel: prometheus_model.LabelValue(eventData.Service + "." + eventData.Project + "-" + stage.Name + "." + gateway + ":80")},
						},
					},
				},
			},
		}
		config.ScrapeConfigs = append(config.ScrapeConfigs, scrapeConfig)

		ag := alertingGroup{
			Name: eventData.Service + " " + eventData.Project + "-" + stage.Name + " alerts",
		}
		for _, objective := range eventData.ServiceObjectives.Objectives {

			indicator := getServiceIndicatorForObjective(objective, eventData.ServiceIndicators)
			if indicator != nil {
				ar := alertingRule{
					Alert: objective.Name,
					Expr:  indicator.Query + " > " + fmt.Sprintf("%f", objective.Threshold),
					For:   objective.Timeframe,
					Labels: alertingLabel{
						Severity: "webhook",
					},
					Annotations: alertingAnnotations{
						Summary:     objective.Name,
						Description: "Pod name {{ $labels.pod_name }}",
					},
				}
				ag.Rules = append(ag.Rules, ar)
			}
		}
		ars.Groups = append(ars.Groups, ag)
	}
	alertingRulesYAMLString, err := yaml.Marshal(ars)
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

func getServiceIndicatorForObjective(objective *models.ServiceObjective, indicators *models.ServiceIndicators) *models.ServiceIndicator {
	for _, indicator := range indicators.Indicators {
		if indicator.Name == objective.Name {
			return indicator
		}
	}
	return nil
}

func deletePrometheusPod() error {

	if err := keptnutils.RestartPodsWithSelector(false, "monitoring", "app=prometheus-server"); err != nil {
		return err
	}
	return nil
}

func storeMonitoringResources(eventData events.ConfigureMonitoringEventData, logger keptnutils.Logger) (*models.Version, error) {
	resources := []*models.Resource{}

	serviceObjectives, err := yaml.Marshal(eventData.ServiceObjectives)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal service objectives. %s", err.Error())
	}
	serviceObjectivesURI := `service-objectives.yaml`
	serviceObjectivesRes := models.Resource{
		ResourceURI:     &serviceObjectivesURI,
		ResourceContent: string(serviceObjectives),
	}

	serviceIndicators, err := yaml.Marshal(eventData.ServiceIndicators)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal service indicators. %s", err.Error())
	}
	serviceIndicatorURI := `service-indicators.yaml`
	serviceIndicatorRes := models.Resource{
		ResourceURI:     &serviceIndicatorURI,
		ResourceContent: string(serviceIndicators),
	}

	remediation, err := yaml.Marshal(eventData.Remediation)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal remediation. %s", err.Error())
	}
	remediationURI := `remediation.yaml`
	remediationRes := models.Resource{
		ResourceURI:     &remediationURI,
		ResourceContent: string(remediation),
	}

	resources = append(resources, &serviceObjectivesRes, &serviceIndicatorRes, &remediationRes)

	return storeResourcesForService(eventData.Project, eventData.Service, resources, logger)
}

// logErrAndRespondWithDoneEvent sends a keptn done event to the keptn eventbroker
func logErrAndRespondWithDoneEvent(event cloudevents.Event, version *models.Version, err error, logger keptnutils.Logger) error {
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

// storeResourcesForService stores the resource for a service using the keptnutils.ResourceHandler
func storeResourcesForService(project string, service string, resources []*models.Resource, logger keptnutils.Logger) (*models.Version, error) {
	configEndpoint, err := utils.GetServiceEndpoint(configservice)
	resourceHandler := keptnutils.NewResourceHandler(configEndpoint.Host)

	// TODO: Use CreateServiceResources(project, service, resources)
	versionStr, err := resourceHandler.CreateServiceResources(project, "dev", service, resources)
	if err != nil {
		return nil, fmt.Errorf("Storing monitoring files failed. %s", err.Error())
	}

	logger.Info("Monitoring files successfully stored")
	version := models.Version{
		Version: versionStr,
	}

	return &version, nil
}
