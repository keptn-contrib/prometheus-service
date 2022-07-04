package eventhandling

import (
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/kelseyhightower/envconfig"
	"github.com/keptn-contrib/prometheus-service/utils"
	"github.com/keptn/go-utils/pkg/lib/keptn"
	keptnv2 "github.com/keptn/go-utils/pkg/lib/v0_2_0"
	"k8s.io/client-go/kubernetes"
)

// PrometheusEventHandler defines a handler for events
type PrometheusEventHandler interface {
	HandleEvent() error
}

// NoOpEventHandler does nothing
type NoOpEventHandler struct {
}

// HandleEvent processes an event
func (e NoOpEventHandler) HandleEvent() error {
	return nil
}

var env utils.EnvConfig

// NewEventHandler creates a new Handler for an incoming event
func NewEventHandler(event cloudevents.Event, logger *keptn.Logger, keptnHandler *keptnv2.Keptn, kubeClient *kubernetes.Clientset, k8sNamespace string) PrometheusEventHandler {
	logger.Debug("Received event: " + event.Type())

	if err := envconfig.Process("", &env); err != nil {
		logger.Error("Failed to process env var: " + err.Error())
	}

	if event.Type() == keptnv2.GetTriggeredEventType(keptnv2.ConfigureMonitoringTaskName) {
		return &ConfigureMonitoringEventHandler{
			logger:       logger,
			event:        event,
			keptnHandler: keptnHandler,
			k8sNamespace: k8sNamespace,
		}
	} else if event.Type() == keptnv2.GetTriggeredEventType(keptnv2.GetSLITaskName) {
		return &GetSliEventHandler{
			event:        event,
			keptnHandler: keptnHandler,
			kubeClient:   kubeClient,
		}
	}

	logger.Error("Unknown event type " + event.Type())

	return &NoOpEventHandler{}

}
