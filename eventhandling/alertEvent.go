package eventhandling

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	keptncommons "github.com/keptn/go-utils/pkg/lib"
	"github.com/nats-io/nats.go"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/keptn/go-utils/pkg/lib/keptn"
	keptnv2 "github.com/keptn/go-utils/pkg/lib/v0_2_0"

	cenats "github.com/cloudevents/sdk-go/protocol/nats/v2"
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
	AlertName  string `json:"alertname,omitempty"`
	Namespace  string `json:"namespace,omitempty"`
	PodName    string `json:"pod_name,omitempty"`
	Severity   string `json:"severity,omitempty"`
	Service    string `json:"service,omitempty" yaml:"service"`
	Stage      string `json:"stage,omitempty" yaml:"stage"`
	Project    string `json:"project,omitempty" yaml:"project"`
	Deployment string `json:"deployment,omitempty" yaml:"deployment"`
}

type annotations struct {
	Summary     string `json:"summary"`
	Description string `json:"descriptions,omitempty"`
}

type remediationTriggeredEventData struct {
	keptnv2.EventData

	// Problem contains details about the problem
	Problem keptncommons.ProblemEventData `json:"problem"`
	// Deployment contains the current deployment, that is inferred from the alert event

	Deployment keptnv2.DeploymentFinishedData `json:"deployment"`
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

	problemData := keptncommons.ProblemEventData{
		State:          problemState,
		ProblemID:      "",
		ProblemTitle:   event.Alerts[0].Annotations.Summary,
		ProblemDetails: json.RawMessage(`{"problemDetails":"` + event.Alerts[0].Annotations.Description + `"}`),
		ProblemURL:     event.Alerts[0].GeneratorURL,
		ImpactedEntity: event.Alerts[0].Labels.PodName,
		Project:        event.Alerts[0].Labels.Project,
		Stage:          event.Alerts[0].Labels.Stage,
		Service:        event.Alerts[0].Labels.Service,
		Labels: map[string]string{
			"deployment": event.Alerts[0].Labels.Deployment,
		},
	}

	newEventData := remediationTriggeredEventData{
		EventData: keptnv2.EventData{
			Project: event.Alerts[0].Labels.Project,
			Stage:   event.Alerts[0].Labels.Stage,
			Service: event.Alerts[0].Labels.Service,
			Labels: map[string]string{
				"Problem URL": event.Alerts[0].GeneratorURL,
			},
		},
		Problem: problemData,
		Deployment: keptnv2.DeploymentFinishedData{
			DeploymentNames: []string{
				event.Alerts[0].Labels.Deployment,
			},
		},
	}

	if event.Alerts[0].Fingerprint != "" {
		// Note: fingerprint is always the same, we will append the startdate to create a unique keptn context
		shkeptncontext = createOrApplyKeptnContext(event.Alerts[0].Fingerprint + event.Alerts[0].StartsAt)
		logger.Debug("shkeptncontext=" + shkeptncontext)
	} else {
		logger.Debug("NO SHKEPTNCONTEXT SET")
	}

	logger.Debug("Sending event to eventbroker")
	err = createAndSendCE(newEventData, shkeptncontext)
	if err != nil {
		logger.Error("Could not send cloud event: " + err.Error())
		rw.WriteHeader(500)
	} else {
		logger.Debug("event successfully dispatched to eventbroker")
		rw.WriteHeader(201)
	}
}

// createAndSendCE create a new problem.triggered event and send it to Keptn
func createAndSendCE(problemData remediationTriggeredEventData, shkeptncontext string) error {
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

	err = forwardEventToNATSServer(event)
	if err != nil {
		return err
	}

	return nil
}

func forwardEventToNATSServer(event cloudevents.Event) error {
	pubSubConnection, err := createPubSubConnection(event.Context.GetType())
	if err != nil {
		return err
	}

	c, err := cloudevents.NewClient(pubSubConnection)
	if err != nil {
		log.Printf("Failed to create cloudevents client: %v", err)
		return err
	}

	cloudevents.WithEncodingStructured(context.Background())

	if result := c.Send(context.Background(), event); cloudevents.IsUndelivered(result) {
		log.Printf("Failed to send cloud event: %v", result.Error())
	} else {
		log.Printf("Sent: %s, accepted: %t", event.ID(), cloudevents.IsACK(result))
	}
	return nil
}

func createPubSubConnection(topic string) (*cenats.Sender, error) {
	if topic == "" {
		return nil, errors.New("no PubSub Topic defined")
	}

	p, err := cenats.NewSender("nats://keptn-nats", topic, cenats.NatsOptions(nats.MaxReconnects(-1)))
	if err != nil {
		log.Printf("Failed to create nats protocol, %v", err)
		return nil, err
	}

	return p, nil
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
