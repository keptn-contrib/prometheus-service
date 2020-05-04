package eventhandling

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	cloudevents "github.com/cloudevents/sdk-go"
	cloudeventsclient "github.com/cloudevents/sdk-go/pkg/cloudevents/client"
	cloudeventshttp "github.com/cloudevents/sdk-go/pkg/cloudevents/transport/http"
	"github.com/cloudevents/sdk-go/pkg/cloudevents/types"
	"github.com/google/uuid"
	"github.com/keptn/go-utils/pkg/lib"

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
	Fingerprint string `json:"fingerprint"`
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

// ProcessAndForwardAlertEvent reads the payload from the request and sends a valid Cloud Event to the keptn event broker
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

	newProblemData := keptn.ProblemEventData{
		State:          problemState,
		ProblemID:      "",
		ProblemTitle:   event.Alerts[0].Annotations.Summary,
		ProblemDetails: json.RawMessage(`{"problemDetails":"` + event.Alerts[0].Annotations.Description + `"}`),
		ImpactedEntity: event.Alerts[0].Labels.PodName,
		Project:        event.Alerts[0].Labels.Project,
		Stage:          event.Alerts[0].Labels.Stage,
		Service:        event.Alerts[0].Labels.Service,
	}

	if event.Alerts[0].Fingerprint != "" {
		shkeptncontext = createOrApplyKeptnContext(event.Alerts[0].Fingerprint)
	}

	logger.Debug("Sending event to eventbroker")
	err = createAndSendCE(eventbroker, newProblemData, shkeptncontext)
	if err != nil {
		logger.Error("Could not send cloud event: " + err.Error())
		rw.WriteHeader(500)
	} else {
		logger.Debug("Event successfully dispatched to eventbroker")
		rw.WriteHeader(201)
	}
}

func createAndSendCE(eventbroker string, problemData keptn.ProblemEventData, shkeptncontext string) error {
	source, _ := url.Parse("prometheus")
	contentType := "application/json"

	endPoint, err := utils.GetServiceEndpoint(eventbroker)

	ce := cloudevents.Event{
		Context: cloudevents.EventContextV02{
			ID:          uuid.New().String(),
			Time:        &types.Timestamp{Time: time.Now()},
			Type:        keptn.ProblemOpenEventType,
			Source:      types.URLRef{URL: *source},
			ContentType: &contentType,
			Extensions:  map[string]interface{}{"shkeptncontext": shkeptncontext},
		}.AsV02(),
		Data: problemData,
	}

	t, err := cloudeventshttp.New(
		cloudeventshttp.WithTarget(endPoint.String()),
		cloudeventshttp.WithEncoding(cloudeventshttp.StructuredV02),
	)
	if err != nil {
		return errors.New("Failed to create transport:" + err.Error())
	}

	c, err := cloudeventsclient.New(t)
	if err != nil {
		return errors.New("Failed to create HTTP client:" + err.Error())
	}

	if _, _, err := c.Send(context.Background(), ce); err != nil {
		return errors.New("Failed to send cloudevent:, " + err.Error())
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
