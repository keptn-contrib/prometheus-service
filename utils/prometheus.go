package utils

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"

	keptncommon "github.com/keptn/go-utils/pkg/lib/keptn"
	alertConfig "github.com/prometheus/alertmanager/config"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"

	keptnv2 "github.com/keptn/go-utils/pkg/lib/v0_2_0"
	"net/http"
)

const Throughput = "throughput"
const ErrorRate = "error_rate"
const RequestLatencyP50 = "response_time_p50"
const RequestLatencyP90 = "response_time_p90"
const RequestLatencyP95 = "response_time_p95"

type prometheusResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric struct {
			} `json:"metric"`
			Value []interface{} `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

// Handler interacts with a prometheus API endpoint
type Handler struct {
	ApiURL        string
	Username      string
	Password      string
	Project       string
	Stage         string
	Service       string
	HTTPClient    *http.Client
	CustomFilters []*keptnv2.SLIFilter
	CustomQueries map[string]string
}

const alertManagerYml = `global:
templates:
- '/etc/alertmanager/*.tmpl'
route:
  receiver: keptn_integration
  group_by: ['alertname', 'priority']
  group_wait: 10s
  repeat_interval: 30m
  routes:
    - receiver: keptn_integration
    # Send severity=webhook alerts to the webhook
      match:
        severity: webhook
      group_wait: 10s
      repeat_interval: 1m

receivers:
- name: keptn_integration
  webhook_configs:
  - url: http://prometheus-service.keptn.svc.cluster.local:8080`

const alertManagerDefaultTemplate = `{{ define "__alertmanager" }}AlertManager{{ end }}
{{ define "__alertmanagerURL" }}{{ .ExternalURL }}/#/alerts?receiver={{ .Receiver }}{{ end }}
{{ define "__subject" }}[{{ .Status | toUpper }}{{ if eq .Status "firing" }}:{{ .Alerts.Firing | len }}{{ end }}] {{ .GroupLabels.SortedPairs.Values | join " " }} {{ if gt (len .CommonLabels) (len .GroupLabels) }}({{ with .CommonLabels.Remove .GroupLabels.Names }}{{ .Values | join " " }}{{ end }}){{ end }}{{ end }}
{{ define "__description" }}{{ end }}
{{ define "__text_alert_list" }}{{ range . }}Labels:
{{ range .Labels.SortedPairs }} - {{ .Name }} = {{ .Value }}
{{ end }}Annotations:
{{ range .Annotations.SortedPairs }} - {{ .Name }} = {{ .Value }}
{{ end }}Source: {{ .GeneratorURL }}
{{ end }}{{ end }}
{{ define "slack.default.title" }}{{ template "__subject" . }}{{ end }}
{{ define "slack.default.username" }}{{ template "__alertmanager" . }}{{ end }}
{{ define "slack.default.fallback" }}{{ template "slack.default.title" . }} | {{ template "slack.default.titlelink" . }}{{ end }}
{{ define "slack.default.pretext" }}{{ end }}
{{ define "slack.default.titlelink" }}{{ template "__alertmanagerURL" . }}{{ end }}
{{ define "slack.default.iconemoji" }}{{ end }}
{{ define "slack.default.iconurl" }}{{ end }}
{{ define "slack.default.text" }}{{ end }}
{{ define "hipchat.default.from" }}{{ template "__alertmanager" . }}{{ end }}
{{ define "hipchat.default.message" }}{{ template "__subject" . }}{{ end }}
{{ define "pagerduty.default.description" }}{{ template "__subject" . }}{{ end }}
{{ define "pagerduty.default.client" }}{{ template "__alertmanager" . }}{{ end }}
{{ define "pagerduty.default.clientURL" }}{{ template "__alertmanagerURL" . }}{{ end }}
{{ define "pagerduty.default.instances" }}{{ template "__text_alert_list" . }}{{ end }}
{{ define "opsgenie.default.message" }}{{ template "__subject" . }}{{ end }}
{{ define "opsgenie.default.description" }}{{ .CommonAnnotations.SortedPairs.Values | join " " }}
{{ if gt (len .Alerts.Firing) 0 -}}
Alerts Firing:
{{ template "__text_alert_list" .Alerts.Firing }}
{{- end }}
{{ if gt (len .Alerts.Resolved) 0 -}}
Alerts Resolved:
{{ template "__text_alert_list" .Alerts.Resolved }}
{{- end }}
{{- end }}
{{ define "opsgenie.default.source" }}{{ template "__alertmanagerURL" . }}{{ end }}
{{ define "victorops.default.message" }}{{ template "__subject" . }} | {{ template "__alertmanagerURL" . }}{{ end }}
{{ define "victorops.default.from" }}{{ template "__alertmanager" . }}{{ end }}
{{ define "email.default.subject" }}{{ template "__subject" . }}{{ end }}
{{ define "email.default.html" }}
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<!--
Style and HTML derived from https://github.com/mailgun/transactional-email-templates
The MIT License (MIT)
Copyright (c) 2014 Mailgun
Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:
The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.
THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
-->
<html xmlns="http://www.w3.org/1999/xhtml" xmlns="http://www.w3.org/1999/xhtml" style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;">
<head style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;">
<meta name="viewport" content="width=device-width" style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;" />
<meta http-equiv="Content-Type" content="text/html; charset=UTF-8" style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;" />
<title style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;">{{ template "__subject" . }}</title>
</head>
<body itemscope="" itemtype="http://schema.org/EmailMessage" style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; -webkit-font-smoothing: antialiased; -webkit-text-size-adjust: none; height: 100%; line-height: 1.6em; width: 100% !important; background-color: #f6f6f6; margin: 0; padding: 0;" bgcolor="#f6f6f6">
<table style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; width: 100%; background-color: #f6f6f6; margin: 0;" bgcolor="#f6f6f6">
  <tr style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;">
    <td style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; vertical-align: top; margin: 0;" valign="top"></td>
    <td width="600" style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; vertical-align: top; display: block !important; max-width: 600px !important; clear: both !important; width: 100% !important; margin: 0 auto; padding: 0;" valign="top">
      <div style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; max-width: 600px; display: block; margin: 0 auto; padding: 0;">
        <table width="100%" cellpadding="0" cellspacing="0" style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; border-radius: 3px; background-color: #fff; margin: 0; border: 1px solid #e9e9e9;" bgcolor="#fff">
          <tr style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;">
            <td style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 16px; vertical-align: top; color: #fff; font-weight: 500; text-align: center; border-radius: 3px 3px 0 0; background-color: #E6522C; margin: 0; padding: 20px;" align="center" bgcolor="#E6522C" valign="top">
              {{ .Alerts | len }} alert{{ if gt (len .Alerts) 1 }}s{{ end }} for {{ range .GroupLabels.SortedPairs }}
                {{ .Name }}={{ .Value }}
              {{ end }}
            </td>
          </tr>
          <tr style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;">
            <td style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; vertical-align: top; margin: 0; padding: 10px;" valign="top">
              <table width="100%" cellpadding="0" cellspacing="0" style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;">
                <tr style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;">
                  <td style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; vertical-align: top; margin: 0; padding: 0 0 20px;" valign="top">
                    <a href="{{ template "__alertmanagerURL" . }}" style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; color: #FFF; text-decoration: none; line-height: 2em; font-weight: bold; text-align: center; cursor: pointer; display: inline-block; border-radius: 5px; text-transform: capitalize; background-color: #348eda; margin: 0; border-color: #348eda; border-style: solid; border-width: 10px 20px;">View in {{ template "__alertmanager" . }}</a>
                  </td>
                </tr>
                {{ if gt (len .Alerts.Firing) 0 }}
                <tr style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;">
                  <td style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; vertical-align: top; margin: 0; padding: 0 0 20px;" valign="top">
                    <strong style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;">[{{ .Alerts.Firing | len }}] Firing</strong>
                  </td>
                </tr>
                {{ end }}
                {{ range .Alerts.Firing }}
                <tr style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;">
                  <td style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; vertical-align: top; margin: 0; padding: 0 0 20px;" valign="top">
                    <strong style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;">Labels</strong><br style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;" />
                    {{ range .Labels.SortedPairs }}{{ .Name }} = {{ .Value }}<br style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;" />{{ end }}
                    {{ if gt (len .Annotations) 0 }}<strong style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;">Annotations</strong><br style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;" />{{ end }}
                    {{ range .Annotations.SortedPairs }}{{ .Name }} = {{ .Value }}<br style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;" />{{ end }}
                    <a href="{{ .GeneratorURL }}" style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; color: #348eda; text-decoration: underline; margin: 0;">Source</a><br style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;" />
                  </td>
                </tr>
                {{ end }}
                {{ if gt (len .Alerts.Resolved) 0 }}
                  {{ if gt (len .Alerts.Firing) 0 }}
                <tr style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;">
                  <td style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; vertical-align: top; margin: 0; padding: 0 0 20px;" valign="top">
                    <br style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;" />
                    <hr style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;" />
                    <br style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;" />
                  </td>
                </tr>
                  {{ end }}
                <tr style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;">
                  <td style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; vertical-align: top; margin: 0; padding: 0 0 20px;" valign="top">
                    <strong style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;">[{{ .Alerts.Resolved | len }}] Resolved</strong>
                  </td>
                </tr>
                {{ end }}
                {{ range .Alerts.Resolved }}
                <tr style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;">
                  <td style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; vertical-align: top; margin: 0; padding: 0 0 20px;" valign="top">
                    <strong style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;">Labels</strong><br style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;" />
                    {{ range .Labels.SortedPairs }}{{ .Name }} = {{ .Value }}<br style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;" />{{ end }}
                    {{ if gt (len .Annotations) 0 }}<strong style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;">Annotations</strong><br style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;" />{{ end }}
                    {{ range .Annotations.SortedPairs }}{{ .Name }} = {{ .Value }}<br style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;" />{{ end }}
                    <a href="{{ .GeneratorURL }}" style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; color: #348eda; text-decoration: underline; margin: 0;">Source</a><br style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;" />
                  </td>
                </tr>
                {{ end }}
              </table>
            </td>
          </tr>
        </table>
        <div style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; width: 100%; clear: both; color: #999; margin: 0; padding: 20px;">
          <table width="100%" style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;">
            <tr style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;">
              <td style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 12px; vertical-align: top; text-align: center; color: #999; margin: 0; padding: 0 0 20px;" align="center" valign="top"><a href="{{ .ExternalURL }}" style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 12px; color: #999; text-decoration: underline; margin: 0;">Sent by {{ template "__alertmanager" . }}</a></td>
            </tr>
          </table>
        </div></div>
    </td>
    <td style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; vertical-align: top; margin: 0;" valign="top"></td>
  </tr>
</table>
</body>
</html>
{{ end }}
{{ define "pushover.default.title" }}{{ template "__subject" . }}{{ end }}
{{ define "pushover.default.message" }}{{ .CommonAnnotations.SortedPairs.Values | join " " }}
{{ if gt (len .Alerts.Firing) 0 }}
Alerts Firing:
{{ template "__text_alert_list" .Alerts.Firing }}
{{ end }}
{{ if gt (len .Alerts.Resolved) 0 }}
Alerts Resolved:
{{ template "__text_alert_list" .Alerts.Resolved }}
{{ end }}
{{ end }}
{{ define "pushover.default.url" }}{{ template "__alertmanagerURL" . }}{{ end }}`

const alertManagerSlackTemplate = `{{ define "slack.devops.text" }}
{{range .Alerts}}{{.Annotations.DESCRIPTION}}
{{end}}alertmanager-templates
{{ end }}`

type PrometheusHelper struct {
	KubeApi *kubernetes.Clientset
}

// NewPrometheusHelper creates a new PrometheusHelper
func NewPrometheusHelper() (*PrometheusHelper, error) {

	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	clientSet, err := kubernetes.NewForConfig(config)

	if err != nil {
		return nil, err
	}

	return &PrometheusHelper{KubeApi: clientSet}, nil
}

func (p *PrometheusHelper) UpdateConfigMap(cm *v1.ConfigMap, namespace string) error {
	_, err := p.KubeApi.CoreV1().ConfigMaps(namespace).Update(cm)
	if err != nil {
		return err
	}

	return nil
}

func (p *PrometheusHelper) GetConfigMap(name string, namespace string) (*v1.ConfigMap, error) {
	return p.KubeApi.CoreV1().ConfigMaps(namespace).Get(name, metav1.GetOptions{})
}

func (p *PrometheusHelper) CreateConfigMap(cm *v1.ConfigMap, namespace string) error {
	_, err := p.KubeApi.CoreV1().ConfigMaps(namespace).Create(cm)
	if err != nil {
		return err
	}

	return nil
}

func (p *PrometheusHelper) DeletePod(label string, namespace string) error {
	pod_list, err := p.KubeApi.CoreV1().Pods(namespace).List(metav1.ListOptions{
		LabelSelector: label,
	})
	if err != nil {
		return err
	}

	for _, pod := range pod_list.Items {
		err := p.KubeApi.CoreV1().Pods(namespace).Delete(pod.Name, &metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *PrometheusHelper) CreateAMTempConfigMap(name string, namespace string) error {
	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string]string{},
	}

	cm.Data["default.tmpl"] = alertManagerDefaultTemplate
	cm.Data["slack.tmpl"] = alertManagerSlackTemplate

	return p.CreateConfigMap(cm, namespace)
}

func (p *PrometheusHelper) UpdateAMConfigMap(name string, filename string, namespace string) error {
	getCM, err := p.GetConfigMap(name, namespace)
	if err != nil {
		return err
	}

	var config alertConfig.Config
	err = yaml.Unmarshal([]byte(getCM.Data[filename]), &config)
	if err != nil {
		return err
	}

	var keptnAlertConfig alertConfig.Config
	err = yaml.Unmarshal([]byte(alertManagerYml), &keptnAlertConfig)
	if err != nil {
		return err
	}

	for _, rec := range config.Receivers {
		if rec.Name == "keptn_integration" {
			return errors.New("keptn_integration reciever is already present")
		}
	}

	for _, route := range config.Route.Routes {
		if route.Receiver == "keptn_integration" {
			return errors.New("keptn_integration reciever is already present in routes")
		}
	}

	config.Receivers = append(config.Receivers, keptnAlertConfig.Receivers...)
	config.Templates = append(config.Templates, keptnAlertConfig.Templates...)
	config.Route.Routes = append(config.Route.Routes, keptnAlertConfig.Route.Routes...)
	getCM.Data[filename] = fmt.Sprint(config)

	return p.UpdateConfigMap(getCM, namespace)
}

// NewPrometheusHandler returns a new prometheus handler that interacts with the Prometheus REST API
func NewPrometheusHandler(apiURL string, project string, stage string, service string, customFilters []*keptnv2.SLIFilter) *Handler {
	ph := &Handler{
		ApiURL:        apiURL,
		Project:       project,
		Stage:         stage,
		Service:       service,
		HTTPClient:    &http.Client{},
		CustomFilters: customFilters,
	}

	return ph
}

// GetSLIValue retrieves the specified value via the Prometheus API
func (ph *Handler) GetSLIValue(metric string, start string, end string, logger keptncommon.LoggerInterface) (float64, error) {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	startUnix, err := parseUnixTimestamp(start)
	if err != nil {
		return 0, err
	}
	endUnix, _ := parseUnixTimestamp(end)
	if err != nil {
		return 0, err
	}
	query, err := ph.getMetricQuery(metric, startUnix, endUnix)
	if err != nil {
		return 0, err
	}
	queryString := ph.ApiURL + "/api/v1/query?query=" + url.QueryEscape(query) + "&time=" + strconv.FormatInt(endUnix.Unix(), 10)
	logger.Info("Generated query: /api/v1/query?query=" + query + "&time=" + strconv.FormatInt(endUnix.Unix(), 10))

	req, err := http.NewRequest("GET", queryString, nil)
	req.Header.Set("Content-Type", "application/json")

	resp, err := ph.HTTPClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return 0, errors.New("metric could not be received")
	}

	prometheusResult := &prometheusResponse{}

	err = json.Unmarshal(body, prometheusResult)
	if err != nil {
		return 0, err
	}

	if len(prometheusResult.Data.Result) == 0 || len(prometheusResult.Data.Result[0].Value) == 0 {
		logger.Info("Prometheus Result is 0, returning value 0")
		// for the error rate query, the result is received with no value if the error rate is 0, so we have to assume that's OK at this point
		return 0, nil
	}

	parsedValue := fmt.Sprintf("%v", prometheusResult.Data.Result[0].Value[1])
	floatValue, err := strconv.ParseFloat(parsedValue, 64)
	logger.Info(fmt.Sprintf("Prometheus Result is %v", floatValue))
	if err != nil {
		return 0, nil
	}
	return floatValue, nil
}

func (ph *Handler) getMetricQuery(metric string, start time.Time, end time.Time) (string, error) {
	query := ph.CustomQueries[metric]
	if query != "" {
		query = ph.replaceQueryParameters(query, start, end)

		return query, nil
	}

	switch metric {
	case Throughput:
		return ph.getThroughputQuery(start, end), nil
	case ErrorRate:
		return ph.getErrorRateQuery(start, end), nil
	case RequestLatencyP50:
		return ph.getRequestLatencyQuery("50", start, end), nil
	case RequestLatencyP90:
		return ph.getRequestLatencyQuery("90", start, end), nil
	case RequestLatencyP95:
		return ph.getRequestLatencyQuery("95", start, end), nil
	default:
		return "", errors.New("unsupported SLI")
	}
}

func (ph *Handler) replaceQueryParameters(query string, start time.Time, end time.Time) string {
	for _, filter := range ph.CustomFilters {
		filter.Value = strings.Replace(filter.Value, "'", "", -1)
		filter.Value = strings.Replace(filter.Value, "\"", "", -1)
		query = strings.Replace(query, "$"+filter.Key, filter.Value, -1)
		query = strings.Replace(query, "$"+strings.ToUpper(filter.Key), filter.Value, -1)
	}
	query = strings.Replace(query, "$PROJECT", ph.Project, -1)
	query = strings.Replace(query, "$STAGE", ph.Stage, -1)
	query = strings.Replace(query, "$SERVICE", ph.Service, -1)
	query = strings.Replace(query, "$project", ph.Project, -1)
	query = strings.Replace(query, "$stage", ph.Stage, -1)
	query = strings.Replace(query, "$service", ph.Service, -1)
	durationString := strconv.FormatInt(getDurationInSeconds(start, end), 10) + "s"

	query = strings.Replace(query, "$DURATION_SECONDS", durationString, -1)
	return query
}

func (ph *Handler) getThroughputQuery(start time.Time, end time.Time) string {
	if ph.CustomQueries != nil && ph.CustomQueries["throughput"] != "" {
		query := ph.CustomQueries["throughput"]
		query = ph.replaceQueryParameters(query, start, end)
		return query
	}
	return ph.getDefaultThroughputQuery(start, end)
}

func (ph *Handler) getDefaultThroughputQuery(start time.Time, end time.Time) string {
	filterExpr := ph.getDefaultFilterExpression()
	durationString := strconv.FormatInt(getDurationInSeconds(start, end), 10) + "s"
	// e.g. sum(rate(http_requests_total{job="carts-sockshop-dev"}[30m]))&time=1571649085
	/*
		{
		    "status": "success",
		    "data": {
		        "resultType": "vector",
		        "result": [
		            {
		                "metric": {},
		                "value": [
		                    1571649085,
		                    "0.20111420612813372"
		                ]
		            }
		        ]
		    }
		}
	*/
	return "sum(rate(http_requests_total{" + filterExpr + "}[" + durationString + "]))"
}

func (ph *Handler) getErrorRateQuery(start time.Time, end time.Time) string {
	if ph.CustomQueries != nil && ph.CustomQueries["error_rate"] != "" {
		query := ph.CustomQueries["error_rate"]
		query = ph.replaceQueryParameters(query, start, end)
		return query
	}
	return ph.getDefaultErrorRateQuery(start, end)
}

func (ph *Handler) getDefaultErrorRateQuery(start time.Time, end time.Time) string {
	filterExpr := ph.getDefaultFilterExpression()
	durationString := strconv.FormatInt(getDurationInSeconds(start, end), 10) + "s"
	// e.g. sum(rate(http_requests_total{job="carts-sockshop-dev",status!~'2..'}[30m]))/sum(rate(http_requests_total{job="carts-sockshop-dev"}[30m]))&time=1571649085
	/*
		with value:
		{
		    "status": "success",
		    "data": {
		        "resultType": "vector",
		        "result": [
		            {
		                "metric": {},
		                "value": [
		                    1571649085,
		                    "1.00505917125441"
		                ]
		            }
		        ]
		    }
		}

		no value (error rate 0):
		{
		    "status": "success",
		    "data": {
		        "resultType": "vector",
		        "result": []
		    }
		}
	*/
	return "sum(rate(http_requests_total{" + filterExpr + ",status!~'2..'}[" + durationString + "]))/sum(rate(http_requests_total{" + filterExpr + "}[" + durationString + "]))"
}

func (ph *Handler) getRequestLatencyQuery(percentile string, start time.Time, end time.Time) string {
	if ph.CustomQueries != nil {
		query := ""
		switch percentile {
		case "50":
			query = ph.CustomQueries["response_time_p50"]
			break
		case "90":
			query = ph.CustomQueries["response_time_p90"]
			break
		case "95":
			query = ph.CustomQueries["response_time_p95"]
			break
		default:
			query = ""
		}
		if query != "" {
			query = ph.replaceQueryParameters(query, start, end)
			return query
		}
	}
	return ph.getDefaultRequestLatencyQuery(start, end, percentile)
}

func (ph *Handler) getDefaultRequestLatencyQuery(start time.Time, end time.Time, percentile string) string {
	filterExpr := ph.getDefaultFilterExpression()
	durationString := strconv.FormatInt(getDurationInSeconds(start, end), 10) + "s"
	// e.g. histogram_quantile(0.95, sum(rate(http_response_time_milliseconds_bucket{job='carts-sockshop-dev'}[30m])) by (le))&time=1571649085
	/*
		{
		    "status": "success",
		    "data": {
		        "resultType": "vector",
		        "result": [
		            {
		                "metric": {},
		                "value": [
		                    1571649085,
		                    "4.607481671642585"
		                ]
		            }
		        ]
		    }
		}
	*/
	return "histogram_quantile(0." + percentile + ",sum(rate(http_response_time_milliseconds_bucket{" + filterExpr + "}[" + durationString + "]))by(le))"
}

func (ph *Handler) getDefaultFilterExpression() string {
	filterExpression := ""
	jobFilterFound := false
	if ph.CustomFilters != nil && len(ph.CustomFilters) > 0 {
		for _, filter := range ph.CustomFilters {
			if filter.Key == "job" {
				jobFilterFound = true
			}
			/* if no operator has been included in the label filter, use exact matching (=), e.g.
			e.g.:
			key: handler
			value: ItemsController
			*/
			if !strings.HasPrefix(filter.Value, "=") && !strings.HasPrefix(filter.Value, "!=") && !strings.HasPrefix(filter.Value, "=~") && !strings.HasPrefix(filter.Value, "!~") {
				filter.Value = strings.Replace(filter.Value, "'", "", -1)
				filter.Value = strings.Replace(filter.Value, "\"", "", -1)
				if filterExpression != "" {
					filterExpression = filterExpression + "," + filter.Key + "='" + filter.Value + "'"
				} else {
					filterExpression = filter.Key + "='" + filter.Value + "'"
				}

			} else {
				/* if a valid operator (=, !=, =~, !~) is prepended to the value, use that one
				e.g.:
				key: handler
				value: !=HealthCheckController

				OR

				key: handler
				value: =~.+ItemsController|.+VersionController
				*/
				filter.Value = strings.Replace(filter.Value, "\"", "'", -1)
				if filterExpression != "" {
					filterExpression = filterExpression + "," + filter.Key + filter.Value
				} else {
					filterExpression = filter.Key + filter.Value
				}
			}
		}
	}
	if !jobFilterFound {
		if filterExpression != "" {
			filterExpression = "job='" + ph.Service + "-" + ph.Project + "-" + ph.Stage + "-canary'" + "," + filterExpression
		} else {
			filterExpression = "job='" + ph.Service + "-" + ph.Project + "-" + ph.Stage + "-canary'"
		}

	}
	return filterExpression
}

func parseUnixTimestamp(timestamp string) (time.Time, error) {
	parsedTime, err := time.Parse(time.RFC3339, timestamp)
	if err == nil {
		return parsedTime, nil
	}

	timestampInt, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return time.Now(), err
	}
	unix := time.Unix(timestampInt, 0)
	return unix, nil
}

func getDurationInSeconds(start, end time.Time) int64 {
	seconds := end.Sub(start).Seconds()
	return int64(math.Ceil(seconds))
}
