package prometheus

import (
	"errors"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"strconv"
	"testing"
	"time"

	prometheusfake "github.com/keptn-contrib/prometheus-service/utils/prometheus/fake"
	prometheusAPI "github.com/prometheus/client_golang/api/prometheus/v1"
	prometheusModel "github.com/prometheus/common/model"
)

func TestHandler_GetSLIValue(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	apiMock := prometheusfake.NewMockAPI(mockCtrl)
	handler := Handler{
		PrometheusAPI: apiMock,
	}

	returnValue := prometheusModel.Vector{
		{
			Value: 0,
		},
	}

	apiMock.EXPECT().Query(gomock.Any(), gomock.Any(), gomock.Any()).Return(returnValue, prometheusAPI.Warnings{}, nil).Times(1)

	startTime := strconv.FormatInt(time.Now().UTC().UnixNano(), 10)
	endTime := strconv.FormatInt(time.Now().UTC().UnixNano(), 10)

	value, err := handler.GetSLIValue(Throughput, startTime, endTime)
	require.NoError(t, err)

	require.Equal(t, (float64)(0), value)
}

func TestHandler_GetSLIValueNoResult(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	apiMock := prometheusfake.NewMockAPI(mockCtrl)
	handler := Handler{
		PrometheusAPI: apiMock,
	}

	returnValue := prometheusModel.Vector{}

	apiMock.EXPECT().Query(gomock.Any(), gomock.Any(), gomock.Any()).Return(returnValue, prometheusAPI.Warnings{}, nil).Times(1)

	startTime := strconv.FormatInt(time.Now().UTC().UnixNano(), 10)
	endTime := strconv.FormatInt(time.Now().UTC().UnixNano(), 10)

	_, err := handler.GetSLIValue(Throughput, startTime, endTime)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNoValues)
}

func TestHandler_GetSLIValueMultipleValues(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	apiMock := prometheusfake.NewMockAPI(mockCtrl)
	handler := Handler{
		PrometheusAPI: apiMock,
	}

	returnValue := prometheusModel.Vector{
		{
			Value: 123,
		},
		{
			Value: 999,
		},
	}
	returnWarnings := prometheusAPI.Warnings{}

	apiMock.EXPECT().Query(gomock.Any(), gomock.Any(), gomock.Any()).Return(returnValue, returnWarnings, nil).Times(1)

	startTime := strconv.FormatInt(time.Now().UTC().UnixNano(), 10)
	endTime := strconv.FormatInt(time.Now().UTC().UnixNano(), 10)

	_, err := handler.GetSLIValue(Throughput, startTime, endTime)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrMultipleValues)
}

func TestHandler_GetSLIValueError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	apiMock := prometheusfake.NewMockAPI(mockCtrl)
	handler := Handler{
		PrometheusAPI: apiMock,
	}

	returnValue := prometheusModel.Vector{}
	returnWarnings := prometheusAPI.Warnings{}
	apiError := errors.New("http Error XXX")

	apiMock.EXPECT().Query(gomock.Any(), gomock.Any(), gomock.Any()).Return(returnValue, returnWarnings, apiError).Times(1)

	startTime := strconv.FormatInt(time.Now().UTC().UnixNano(), 10)
	endTime := strconv.FormatInt(time.Now().UTC().UnixNano(), 10)

	_, err := handler.GetSLIValue(Throughput, startTime, endTime)
	require.Error(t, err)
	require.ErrorIs(t, err, apiError)
}
