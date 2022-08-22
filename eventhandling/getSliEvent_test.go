package eventhandling

import (
	"encoding/json"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/golang/mock/gomock"
	prometheusUtils "github.com/keptn-contrib/prometheus-service/utils/prometheus"
	prometheusfake "github.com/keptn-contrib/prometheus-service/utils/prometheus/fake"
	keptnv2 "github.com/keptn/go-utils/pkg/lib/v0_2_0"
	prometheusAPI "github.com/prometheus/client_golang/api/prometheus/v1"
	prometheusModel "github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"math/rand"
	"testing"
)

const eventJSON = `
{
  "data": {
    "deployment": "canary",
    "get-sli": {
      "end": "2022-04-06T14:36:19.667Z",
      "sliProvider": "prometheus",
      "start": "2022-04-06T14:35:03.762Z",
	  "indicators": ["throughput"]
    },
    "project": "sockshop",
    "service": "carts",
    "stage": "staging"
  },
  "gitcommitid": "c8a40997599180a338d72504541c00057550a3dc",
  "id": "585cb332-7198-4605-a0ef-28199268b91d",
  "shkeptncontext": "37a580f4-96ef-4594-b62a-1235b91ed7f6",
  "shkeptnspecversion": "0.2.4",
  "source": "lighthouse-service",
  "specversion": "1.0",
  "time": "2022-04-06T14:36:19.887Z",
  "type": "sh.keptn.event.get-sli.triggered"
}
`

func Test_retrieveMetrics(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	incomingEvent := &cloudevents.Event{}

	err := json.Unmarshal([]byte(eventJSON), incomingEvent)
	require.NoError(t, err)

	eventData := &keptnv2.GetSLITriggeredEventData{}
	err = incomingEvent.DataAs(eventData)
	require.NoError(t, err)

	apiMock := prometheusfake.NewMockAPI(mockCtrl)
	handler := prometheusUtils.Handler{
		Project:       eventData.Project,
		Stage:         eventData.Stage,
		Service:       eventData.Service,
		PrometheusAPI: apiMock,
	}

	sliValue := rand.Float64()
	returnValue := prometheusModel.Vector{
		{
			Value: prometheusModel.SampleValue(sliValue),
		},
	}

	apiMock.EXPECT().Query(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(
		returnValue, prometheusAPI.Warnings{}, nil,
	)

	sliResults := retrieveMetrics(&handler, eventData)

	assert.Len(t, sliResults, 1)
	assert.Contains(t, sliResults, &keptnv2.SLIResult{
		Metric:        prometheusUtils.Throughput,
		Value:         sliValue,
		ComparedValue: 0,
		Success:       true,
		Message:       "",
	})
}

func Test_retrieveMetricsWithMultipleValues(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	incomingEvent := &cloudevents.Event{}

	err := json.Unmarshal([]byte(eventJSON), incomingEvent)
	require.NoError(t, err)

	eventData := &keptnv2.GetSLITriggeredEventData{}
	err = incomingEvent.DataAs(eventData)
	require.NoError(t, err)

	apiMock := prometheusfake.NewMockAPI(mockCtrl)
	handler := prometheusUtils.Handler{
		Project:       eventData.Project,
		Stage:         eventData.Stage,
		Service:       eventData.Service,
		PrometheusAPI: apiMock,
	}

	returnValue := prometheusModel.Vector{
		{
			Value: prometheusModel.SampleValue(8.12830),
		},
		{
			Value: prometheusModel.SampleValue(0.28384),
		},
	}

	apiMock.EXPECT().Query(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(
		returnValue, prometheusAPI.Warnings{}, nil,
	)

	sliResults := retrieveMetrics(&handler, eventData)

	assert.Len(t, sliResults, 1)
	assert.Contains(t, sliResults, &keptnv2.SLIResult{
		Metric:        prometheusUtils.Throughput,
		Value:         0,
		ComparedValue: 0,
		Success:       false,
		Message:       prometheusUtils.ErrMultipleValues.Error(),
	})
}

func Test_retrieveMetricsWithNoValue(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	incomingEvent := &cloudevents.Event{}

	err := json.Unmarshal([]byte(eventJSON), incomingEvent)
	require.NoError(t, err)

	eventData := &keptnv2.GetSLITriggeredEventData{}
	err = incomingEvent.DataAs(eventData)
	require.NoError(t, err)

	apiMock := prometheusfake.NewMockAPI(mockCtrl)
	handler := prometheusUtils.Handler{
		Project:       eventData.Project,
		Stage:         eventData.Stage,
		Service:       eventData.Service,
		PrometheusAPI: apiMock,
	}

	apiMock.EXPECT().Query(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(
		prometheusModel.Vector{}, prometheusAPI.Warnings{}, nil,
	)

	sliResults := retrieveMetrics(&handler, eventData)

	assert.Len(t, sliResults, 1)
	assert.Contains(t, sliResults, &keptnv2.SLIResult{
		Metric:        prometheusUtils.Throughput,
		Value:         0,
		ComparedValue: 0,
		Success:       false,
		Message:       prometheusUtils.ErrNoValues.Error(),
	})
}
