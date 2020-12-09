package eventhandling

import (
	"encoding/json"
	"fmt"
	keptnevents "github.com/keptn/go-utils/pkg/lib"
	"github.com/keptn/go-utils/pkg/lib/keptn"
	keptnv2 "github.com/keptn/go-utils/pkg/lib/v0_2_0"
	"net/http"
	"net/url"
	"strings"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"

	"github.com/keptn-contrib/prometheus-service/utils"
)

type alertManagerEvent struct {
	Receiver string  `json:"receiver"`
	Status   string  `json:"status"`
	Alerts   []alert `json:"alerts""`
}

type alert struct {
	Status      string      `json:"status"`
	Labels      labels      `json:"labels"`
	Annotations annotations `json:"annotations"`
	//StartsAt time   `json:"startsAt"`
	//EndsAt   time   `json:"endsAt"`
	Fingerprint  string `json:"fingerprint"`
	GeneratorURL string `json:"generatorURL"`
}

type labels struct {
	AlertName string `json:"alertname,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	PodName   string `json:"pod_name,omitempty"`
	Severity  string `json:"severity,omitempty"`
	Service   string `json:"service,omitempty" yaml:"service"`
	Stage     string `json:"stage,omitempty" yaml:"stage"`
	Project   string `json:"project,omitempty" yaml:"project"`
}

type annotations struct {
	Summary     string `json:"summary"`
	Description string `json:"descriptions,omitempty"`
}

// ProcessAndForwardAlertEvent reads the payload from the request and sends a valid Cloud event to the keptn event broker
func ProcessAndForwardAlertEvent(rw http.ResponseWriter, requestBody []byte, logger *keptn.Logger, shkeptncontext string) {
	var event alertManagerEvent
	logger.Info("Received alert from Prometheus Alertmanager:" + string(requestBody))
	err := json.Unmarshal(requestBody, &event)
	if err != nil {
		logger.Error("Could not map received event to datastructure: " + err.Error())
		return
	}

	problemState := ""
	if event.Status == "firing" {
		problemState = "OPEN"
	} else if event.Status == "resolved" {
		logger.Info("Don't forward resolved problem.")
		return
	}

	newProblemData := keptnevents.ProblemEventData{
		State:          problemState,
		ProblemID:      "",
		ProblemTitle:   event.Alerts[0].Annotations.Summary,
		ProblemDetails: json.RawMessage(`{"problemDetails":"` + event.Alerts[0].Annotations.Description + `"}`),
		ProblemURL:     event.Alerts[0].GeneratorURL,
		ImpactedEntity: event.Alerts[0].Labels.PodName,
		Project:        event.Alerts[0].Labels.Project,
		Stage:          event.Alerts[0].Labels.Stage,
		Service:        event.Alerts[0].Labels.Service,
	}

	if event.Alerts[0].Fingerprint != "" {
		shkeptncontext = createOrApplyKeptnContext(event.Alerts[0].Fingerprint)
	}

	logger.Debug("Sending event to eventbroker")
	err = createAndSendCE(newProblemData, shkeptncontext)
	if err != nil {
		logger.Error("Could not send cloud event: " + err.Error())
		rw.WriteHeader(500)
	} else {
		logger.Debug("event successfully dispatched to eventbroker")
		rw.WriteHeader(201)
	}
}

func createAndSendCE(problemData keptnevents.ProblemEventData, shkeptncontext string) error {
	source, _ := url.Parse("prometheus")

	eventBrokerURL, err := utils.GetEventBrokerURL()

	event := cloudevents.NewEvent()
	event.SetID(uuid.New().String())
	event.SetTime(time.Now())
	event.SetType(keptnevents.ProblemEventType)
	event.SetSource(source.String())
	event.SetExtension("shkeptncontext", shkeptncontext)
	event.SetDataContentType(cloudevents.ApplicationJSON)
	event.SetData(cloudevents.ApplicationJSON, problemData)

	keptnHandler, err := keptnv2.NewKeptn(&event, keptn.KeptnOpts{
		EventBrokerURL: eventBrokerURL,
	})
	if err != nil {
		return fmt.Errorf("could not initialize Keptn Handler: %s", err.Error())
	}

	if err := keptnHandler.SendCloudEvent(event); err != nil {
		return fmt.Errorf("could not send event: %s", err.Error())
	}

	return nil
}

func createOrApplyKeptnContext(contextID string) string {
	uuid.SetRand(nil)
	keptnContext := uuid.New().String()
	if contextID != "" {
		_, err := uuid.Parse(contextID)
		if err != nil {
			if len(contextID) < 16 {
				paddedContext := fmt.Sprintf("%-16v", contextID)
				uuid.SetRand(strings.NewReader(paddedContext))
			} else {
				uuid.SetRand(strings.NewReader(contextID))
			}

			keptnContext = uuid.New().String()
			uuid.SetRand(nil)
		} else {
			keptnContext = contextID
		}
	}
	return keptnContext
}
