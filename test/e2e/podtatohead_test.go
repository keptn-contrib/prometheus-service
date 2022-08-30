package e2e

import (
	"bytes"
	"context"
	"fmt"
	api "github.com/keptn/go-utils/pkg/api/utils"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/keptn/go-utils/pkg/api/models"
	"github.com/stretchr/testify/require"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPodtatoheadEvaluation(t *testing.T) {
	if !isE2ETestingAllowed() {
		t.Skip("Skipping TestHelloWorldDeployment, not allowed by environment")
	}

	// Setup the E2E test environment
	testEnv, err := newTestEnvironment(
		"../events/podtatohead.deploy-v0.1.1.triggered.json",
		"../shipyard/podtatohead.deployment.yaml",
		"../data/podtatohead.jes-config.yaml",
	)

	require.NoError(t, err)

	additionalResources := []struct {
		FilePath     string
		ResourceName string
	}{
		{FilePath: "../data/podtatoserver-0.1.0.tgz", ResourceName: fmt.Sprintf("charts/%s.tgz", testEnv.EventData.Service)},
		{FilePath: "../data/locust.basic.py", ResourceName: "locust/basic.py"},
		{FilePath: "../data/locust.conf", ResourceName: "locust/locust.conf"},
		{FilePath: "../data/podtatohead.sli.yaml", ResourceName: "prometheus/sli.yaml"},
		{FilePath: "../data/podtatohead.slo.yaml", ResourceName: "slo.yaml"},
		{FilePath: "../data/podtatohead.remediation.yaml", ResourceName: "remediation.yaml"},
	}

	err = testEnv.SetupTestEnvironment()
	require.NoError(t, err)

	// Make sure project is delete after the tests are completed
	// defer testEnv.Cleanup()

	// Upload additional resources to the keptn project
	for _, resource := range additionalResources {
		content, err := ioutil.ReadFile(resource.FilePath)
		require.NoError(t, err, "Unable to read file %s", resource.FilePath)

		err = testEnv.API.AddServiceResource(testEnv.EventData.Project, testEnv.EventData.Stage,
			testEnv.EventData.Service, resource.ResourceName, string(content))

		require.NoErrorf(t, err, "unable to create file %s", resource.ResourceName)
	}

	// Test if the configuration of prometheus was without errors
	t.Run("Configure Prometheus", func(t *testing.T) {
		// Configure monitoring
		configureMonitoring, err := readKeptnContextExtendedCE("../events/podtatohead.configure-monitoring.json")
		require.NoError(t, err)

		configureMonitoringContext, err := testEnv.API.SendEvent(configureMonitoring)
		require.NoError(t, err)

		// wait until prometheus is configured correctly ...
		requireWaitForEvent(t,
			testEnv.API,
			5*time.Minute,
			1*time.Second,
			configureMonitoringContext,
			"sh.keptn.event.configure-monitoring.finished",
			func(event *models.KeptnContextExtendedCE) bool {
				responseEventData, err := parseKeptnEventData(event)
				require.NoError(t, err)

				return responseEventData.Result == "pass" && responseEventData.Status == "succeeded"
			},
			"prometheus-service",
		)

		prometheusNamespace := os.Getenv("PROMETHEUS_NAMESPACE")
		// TODO: Improve checking of prometheus configuration
		// Note: We don't parse and check the configuration at this point, but just making sure that things we write to
		//       the prometheus.yml file are contained in there. Easiest way is to verify that the job configuration is
		//       present and the targets are contained in the content ...
		prometheusConfigMap, err := testEnv.K8s.CoreV1().ConfigMaps(prometheusNamespace).Get(
			context.Background(), "prometheus-server", metav1.GetOptions{},
		)
		require.NoError(t, err)

		prometheusYaml := prometheusConfigMap.Data["prometheus.yml"]
		require.Contains(t, prometheusYaml, "job_name: podtatoserver-e2e-project-staging")
		require.Contains(t, prometheusYaml, "job_name: podtatoserver-e2e-project-staging-canary")
		require.Contains(t, prometheusYaml, "job_name: podtatoserver-e2e-project-staging-primary")
		require.Contains(t, prometheusYaml, "podtatoserver.e2e-project-staging:80")

		alertmanagerConfigMap, err := testEnv.K8s.CoreV1().ConfigMaps(prometheusNamespace).Get(
			context.Background(), "prometheus-alertmanager", metav1.GetOptions{},
		)
		require.NoError(t, err)

		alertmanagerYaml := alertmanagerConfigMap.Data["alertmanager.yml"]
		require.Contains(t, alertmanagerYaml, "name: keptn_integration")
		require.Contains(t, alertmanagerYaml, "receiver: keptn_integration")
		require.Contains(t, alertmanagerYaml, "url: http://prometheus-service.keptn.svc.cluster.local:8080")
	})

	// Test deployment of podtatohead v0.1.1 where all SLI values must be according to SLO
	t.Run("Deploy podtatohead v0.1.1", func(t *testing.T) {
		// Send the event to keptn to deploy, test and evaluate the service
		keptnContext, err := testEnv.API.SendEvent(testEnv.Event)
		require.NoError(t, err)

		// Checking a .started event is received from the evaluation process
		requireWaitForEvent(t,
			testEnv.API,
			5*time.Minute,
			1*time.Second,
			keptnContext,
			"sh.keptn.event.get-sli.started",
			func(_ *models.KeptnContextExtendedCE) bool {
				return true
			},
			"prometheus-service",
		)

		requireWaitForEvent(t,
			testEnv.API,
			5*time.Minute,
			1*time.Second,
			keptnContext,
			"sh.keptn.event.get-sli.finished",
			func(event *models.KeptnContextExtendedCE) bool {
				responseEventData, err := parseKeptnEventData(event)
				require.NoError(t, err)

				return responseEventData.Result == "pass" && responseEventData.Status == "succeeded"
			},
			"prometheus-service",
		)

		requireWaitForEvent(t,
			testEnv.API,
			1*time.Minute,
			1*time.Second,
			keptnContext,
			"sh.keptn.event.evaluation.finished",
			func(event *models.KeptnContextExtendedCE) bool {
				responseEventData, err := parseKeptnEventData(event)
				require.NoError(t, err)

				return responseEventData.Result == "pass" && responseEventData.Status == "succeeded"
			},
			"lighthouse-service",
		)
	})

	// Test deployment of podtatohead v0.1.2 where the lighthouse-service will fail the evaluation
	t.Run("Deploy podtatohead v0.1.2", func(t *testing.T) {
		event, err := readKeptnContextExtendedCE("../events/podtatohead.deploy-v0.1.2.triggered.json")
		require.NoError(t, err)

		keptnContext, err := testEnv.API.SendEvent(event)
		require.NoError(t, err)

		// Checking a .started event is received from the evaluation process
		requireWaitForEvent(t,
			testEnv.API,
			5*time.Minute,
			1*time.Second,
			keptnContext,
			"sh.keptn.event.get-sli.started",
			func(_ *models.KeptnContextExtendedCE) bool {
				return true
			},
			"prometheus-service",
		)

		requireWaitForEvent(t,
			testEnv.API,
			5*time.Minute,
			1*time.Second,
			keptnContext,
			"sh.keptn.event.get-sli.finished",
			func(event *models.KeptnContextExtendedCE) bool {
				responseEventData, err := parseKeptnEventData(event)
				require.NoError(t, err)

				return responseEventData.Result == "pass" && responseEventData.Status == "succeeded"
			},
			"prometheus-service",
		)

		requireWaitForEvent(t,
			testEnv.API,
			1*time.Minute,
			1*time.Second,
			keptnContext,
			"sh.keptn.event.evaluation.finished",
			func(event *models.KeptnContextExtendedCE) bool {
				responseEventData, err := parseKeptnEventData(event)
				require.NoError(t, err)

				return responseEventData.Result == "fail" && responseEventData.Status == "succeeded"
			},
			"lighthouse-service",
		)
	})

	// Note: This part should be improved:
	t.Run("Test Alertmanager", func(t *testing.T) {

		// First create a portforward from the prometheus-service pod to the host (:11111)
		config, err := BuildK8sConfig()
		require.NoError(t, err)

		roundTripper, upgrader, err := spdy.RoundTripperFor(config)
		require.NoError(t, err)

		labelSelector := labels.NewSelector()
		labelRequirement, _ := labels.NewRequirement("app.kubernetes.io/name", selection.Equals, []string{"prometheus-service"})
		labelSelector = labelSelector.Add(*labelRequirement)

		pods, err := testEnv.K8s.CoreV1().Pods(testEnv.Namespace).List(context.TODO(), metav1.ListOptions{
			TypeMeta:      metav1.TypeMeta{},
			LabelSelector: labelSelector.String(),
		})
		require.NoError(t, err)
		require.Len(t, pods.Items, 1)

		path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", testEnv.Namespace, pods.Items[0].Name)
		hostIP := strings.TrimLeft(config.Host, "https:/")
		serverURL := url.URL{Scheme: "https", Path: path, Host: hostIP}

		dailer := spdy.NewDialer(upgrader, &http.Client{Transport: roundTripper}, http.MethodPost, &serverURL)

		stopChan, readyChan := make(chan struct{}, 1), make(chan struct{}, 1)
		out, errOut := new(bytes.Buffer), new(bytes.Buffer)

		forwarder, err := portforward.New(dailer, []string{"11111:8080"}, stopChan, readyChan, out, errOut)
		require.NoError(t, err)

		// If the connection is read, post the prometheus event to the prometheus-service
		go func() {
			for range readyChan {
			}

			requestBody, err := os.Open("../events/prometheus-firing.json")
			require.NoError(t, err)

			_, err = http.DefaultClient.Post("http://localhost:11111", "application/json", requestBody)
			require.NoError(t, err)

			close(stopChan)
		}()

		err = forwarder.ForwardPorts()
		require.NoError(t, err)

		// Check if the following event are in the project:
		//   - remediation.triggered		(making sure prometheus-service actually sends a message to Keptn)
		//   - get-action.finished          (The message format is compatible with remediation-service)
		//   - evaluation.triggered         (making sure Keptn emits an evaluation.triggered)
		//   - get-sli.finished             (making sure prometheus-service responds with data)
		requireWaitForFilteredEvent(t,
			testEnv.API,
			1*time.Minute,
			1*time.Second,
			&api.EventFilter{
				Project:   testEnv.EventData.Project,
				Stage:     testEnv.EventData.Stage,
				Service:   testEnv.EventData.Service,
				EventType: "sh.keptn.event.staging.remediation.triggered",
			},
			func(event *models.KeptnContextExtendedCE) bool {
				return true
			},
			"prometheus",
		)

		requireWaitForFilteredEvent(t,
			testEnv.API,
			1*time.Minute,
			1*time.Second,
			&api.EventFilter{
				Project:   testEnv.EventData.Project,
				Stage:     testEnv.EventData.Stage,
				Service:   testEnv.EventData.Service,
				EventType: "sh.keptn.event.get-action.finished",
			},
			func(event *models.KeptnContextExtendedCE) bool {
				responseEventData, err := parseKeptnEventData(event)
				require.NoError(t, err)

				return responseEventData.Result == "pass" && responseEventData.Status == "succeeded"
			},
			"remediation-service",
		)

		// wait roughly 60 seconds for evaluation.triggered
		requireWaitForFilteredEvent(t,
			testEnv.API,
			2*time.Minute,
			1*time.Second,
			&api.EventFilter{
				Project:   testEnv.EventData.Project,
				Stage:     testEnv.EventData.Stage,
				Service:   testEnv.EventData.Service,
				EventType: "sh.keptn.event.evaluation.triggered",
			},
			func(event *models.KeptnContextExtendedCE) bool {
				return true
			},
			"shipyard-controller",
		)

		// wait for prometheus-service returning a get-sli.finished
		requireWaitForFilteredEvent(t,
			testEnv.API,
			1*time.Minute,
			1*time.Second,
			&api.EventFilter{
				Project:   testEnv.EventData.Project,
				Stage:     testEnv.EventData.Stage,
				Service:   testEnv.EventData.Service,
				EventType: "sh.keptn.event.get-sli.finished",
			},
			func(event *models.KeptnContextExtendedCE) bool {
				responseEventData, err := parseKeptnEventData(event)
				require.NoError(t, err)

				return responseEventData.Result == "pass" && responseEventData.Status == "succeeded"
			},
			"prometheus-service",
		)
	})
}
