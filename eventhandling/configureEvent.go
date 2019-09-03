package eventhandling

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"gopkg.in/yaml.v2"

	cloudevents "github.com/cloudevents/sdk-go"
	"github.com/cloudevents/sdk-go/pkg/cloudevents/client"
	cloudeventshttp "github.com/cloudevents/sdk-go/pkg/cloudevents/transport/http"
	"github.com/cloudevents/sdk-go/pkg/cloudevents/types"

	"github.com/google/uuid"

	"github.com/keptn-contrib/prometheus-service/utils"

	"github.com/keptn/go-utils/pkg/events"
	"github.com/keptn/go-utils/pkg/models"
	keptnutils "github.com/keptn/go-utils/pkg/utils"
)

const configservice = "CONFIGURATION_SERVICE"
const eventbroker = "EVENTBROKER"
const api = "API"

type doneEventData struct {
	Result  string `json:"result"`
	Message string `json:"message"`
	Version string `json:"version"`
}

func GotEvent(ctx context.Context, event cloudevents.Event) error {
	var shkeptncontext string
	event.Context.ExtensionAs("shkeptncontext", &shkeptncontext)

	logger := keptnutils.NewLogger(shkeptncontext, event.Context.GetID(), "prometheus-service")

	// open websocket connection to api component
	/*
		endPoint, err := utils.GetServiceEndpoint(api)
		if err != nil {
			return err
		}

		if endPoint.Host == "" {
			const errorMsg = "Host of api not set"
			logger.Error(errorMsg)
			return errors.New(errorMsg)
		}

		connData := &websockethelper.ConnectionData{}
		if err := event.DataAs(connData); err != nil {
			logger.Error(fmt.Sprintf("Data of the event is incompatible. %s", err.Error()))
			return err
		}

		ws, _, err := websocketutil.OpenWS(*connData, endPoint)
		if err != nil {
			logger.Error(fmt.Sprintf("Opening websocket connection failed. %s", err.Error()))
			return err
		}
		defer ws.Close()
	*/

	// process event
	if event.Type() == events.ConfigureMonitoringEventType {
		version, err := configurePrometheusAndStoreResources(event, *logger)
		if err := logErrAndRespondWithDoneEvent(event, version, err, *logger); err != nil {
			return err
		}

		return nil
	}

	const errorMsg = "Received unexpected keptn event that cannot be processed"
	/*
		if err := websocketutil.WriteWSLog(ws, createEventCopy(event, "sh.keptn.events.log"), errorMsg, true, "INFO"); err != nil {
			logger.Error(fmt.Sprintf("Could not write log to websocket. %s", err.Error()))
		}
	*/
	logger.Error(errorMsg)
	return errors.New(errorMsg)
}

// configurePrometheusAndStoreResources
func configurePrometheusAndStoreResources(event cloudevents.Event, logger keptnutils.Logger) (*models.Version, error) {
	eventData := &events.ConfigureMonitoringEventData{}

	// (1) if prometheus is not installed - install prometheus

	// (2) update config map with alert rule

	// (3) store resources
	return storeMonitoringResources(*eventData, logger)
}

func storeMonitoringResources(eventData events.ConfigureMonitoringEventData, logger keptnutils.Logger) (*models.Version, error) {
	serviceObjectives, err := yaml.Marshal(eventData.ServiceObjectives)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal service objectives. %s", err.Error())
	}
	storeResourceForService(eventData.Service, "service-objectives.yaml", string(serviceObjectives), logger)

	serviceIndicators, err := yaml.Marshal(eventData.ServiceIndicators)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal service indicators. %s", err.Error())
	}
	storeResourceForService(eventData.Service, "service-indicators.yaml", string(serviceIndicators), logger)

	remediation, err := yaml.Marshal(eventData.Remediation)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal remediation. %s", err.Error())
	}

	return storeResourceForService(eventData.Service, "remedation.yaml", string(remediation), logger)
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

	/*
		if err := websocketutil.WriteWSLog(ws, createEventCopy(event, "sh.keptn.events.log"), webSocketMessage, true, "INFO"); err != nil {
			logger.Error(fmt.Sprintf("Could not write log to websocket. %s", err.Error()))
		}
	*/
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

// storeResourceForService stores the resource for a service using the keptnutils.ResourceHandler
func storeResourceForService(service string, resourceURI string, resourceContent string, logger keptnutils.Logger) (*models.Version, error) {
	resource := models.Resource{
		ResourceURI:     &resourceURI,
		ResourceContent: resourceContent,
	}
	resources := []*models.Resource{&resource}

	eventURL, err := utils.GetServiceEndpoint(configservice)
	resourceHandler := keptnutils.NewResourceHandler(eventURL.Host)

	versionStr, err := resourceHandler.CreateProjectResources(service, resources)
	if err != nil {
		return nil, fmt.Errorf("Storing %s file failed. %s", resourceURI, err.Error())
	}

	logger.Info(fmt.Sprintf("Resource %s successfully stored", resourceURI))
	version := models.Version{
		Version: versionStr,
	}

	return &version, nil
}
