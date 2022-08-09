package eventhandling

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/keptn/go-utils/pkg/lib/keptn"
	keptnv2 "github.com/keptn/go-utils/pkg/lib/v0_2_0"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
)

const remediationTaskName = "remediation"

type alertManagerEvent struct {
	Receiver string  `json:"receiver"`
	Status   string  `json:"status"`
	Alerts   []alert `json:"alerts"`
}

// alert coming from prometheus
type alert struct {
	Status       string      `json:"status"`
	Labels       labels      `json:"labels"`
	Annotations  annotations `json:"annotations"`
	StartsAt     string      `json:"startsAt"`
	EndsAt       string      `json:"endsAt"`
	Fingerprint  string      `json:"fingerprint"`
	GeneratorURL string      `json:"generatorURL"`
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

type problemEventData struct {
	Project string                 `json:"project"`
	Stage   string                 `json:"stage"`
	Service string                 `json:"service"`
	Problem keptnv2.ProblemDetails `json:"problem"`
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

	if event.Status == "resolved" {
		logger.Info("Don't forward resolved problem.")
		return
	}

	problemDetails := keptnv2.ProblemDetails{
		ProblemTitle: event.Alerts[0].Labels.AlertName,
		RootCause:    event.Alerts[0].Annotations.Summary,
	}

	newEventData := &problemEventData{
		Project: event.Alerts[0].Labels.Project,
		Stage:   event.Alerts[0].Labels.Stage,
		Service: event.Alerts[0].Labels.Service,
		Problem: problemDetails,
	}

	if event.Alerts[0].Fingerprint != "" {
		// Note: fingerprint is always the same, we will append the startdate to create a unique keptn context
		shkeptncontext = createOrApplyKeptnContext(event.Alerts[0].Fingerprint + event.Alerts[0].StartsAt)
		logger.Debug("shkeptncontext=" + shkeptncontext)
	} else {
		logger.Debug("NO SHKEPTNCONTEXT SET")
	}

	logger.Debug("Sending event to eventbroker")
	err = createAndSendCE(*newEventData, shkeptncontext)
	if err != nil {
		logger.Error("Could not send cloud event: " + err.Error())
		rw.WriteHeader(500)
	} else {
		logger.Debug("event successfully dispatched to eventbroker")
		rw.WriteHeader(201)
	}
}

// createAndSendCE create a new problem.triggered event and send it to Keptn
func createAndSendCE(problemData problemEventData, shkeptncontext string) error {
	source, _ := url.Parse("prometheus")

	eventType := keptnv2.GetTriggeredEventType(problemData.Stage + "." + remediationTaskName)

	event := cloudevents.NewEvent()
	event.SetID(uuid.New().String())
	event.SetTime(time.Now())
	event.SetType(eventType)
	event.SetSource(source.String())
	event.SetDataContentType(cloudevents.ApplicationJSON)
	event.SetExtension("shkeptncontext", shkeptncontext)
	err := event.SetData(cloudevents.ApplicationJSON, problemData)
	if err != nil {
		return fmt.Errorf("unable to set cloud event data: %w", err)
	}

	keptnHandler, err := keptnv2.NewKeptn(&event, keptn.KeptnOpts{})
	if err != nil {
		return fmt.Errorf("could not initialize Keptn Handler: %s", err.Error())
	}

	if err := keptnHandler.SendCloudEvent(event); err != nil {
		return fmt.Errorf("could not send event: %s", err.Error())
	}

	return nil
}

// createOrApplyKeptnContext re-uses the existing Keptn Context or creates a new one based on prometheus fingerprint
func createOrApplyKeptnContext(contextID string) string {
	uuid.SetRand(nil)
	keptnContext := uuid.New().String()
	if contextID != "" {
		_, err := uuid.Parse(contextID)
		if err != nil {
			if len(contextID) < 16 {
				// use provided contxtId as a seed
				paddedContext := fmt.Sprintf("%-16v", contextID)
				uuid.SetRand(strings.NewReader(paddedContext))
			} else {
				// convert hash of contextID
				h := sha256.New()
				h.Write([]byte(contextID))
				bs := h.Sum(nil)

				uuid.SetRand(strings.NewReader(string(bs)))
			}

			keptnContext = uuid.New().String()
			uuid.SetRand(nil)
		} else {
			keptnContext = contextID
		}
	}
	return keptnContext
}
