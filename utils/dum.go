package utils
//
//import (
//	"gopkg.in/yaml.v2"
//	appsv1 "k8s.io/api/apps/v1"
//	v1 "k8s.io/api/core/v1"
//	"k8s.io/api/rbac/v1beta1"
//	"k8s.io/apimachinery/pkg/util/intstr"
//	"k8s.io/client-go/kubernetes"
//	"k8s.io/client-go/rest"
//
//	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
//)
//
//const alertManagerYml = `global:
//templates:
//- '/etc/alertmanager/*.tmpl'
//route:
// receiver: keptn_integration
// group_by: ['alertname', 'priority']
// group_wait: 10s
// repeat_interval: 30m
// routes:
//   - receiver: keptn_integration
//   # Send severity=webhook alerts to the webhook
//     match:
//       severity: webhook
//     group_wait: 10s
//     repeat_interval: 1m
//
//receivers:
//- name: keptn_integration
// webhook_configs:
// - url: http://prometheus-service.keptn.svc.cluster.local:8080`
//
//const prometheusYml = `global:
// scrape_interval: 5s
// evaluation_interval: 5s
//rule_files:
// - /etc/prometheus/prometheus.rules
//alerting:
// alertmanagers:
// - scheme: http
//   static_configs:
//   - targets:
//     - "alertmanager.monitoring.svc:9093"
//
//scrape_configs:
// - job_name: 'kubernetes-apiservers'
//
//   kubernetes_sd_configs:
//   - role: endpoints
//   scheme: https
//
//   tls_config:
//     ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
//   bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token
//
//   relabel_configs:
//   - source_labels: [__meta_kubernetes_namespace, __meta_kubernetes_service_name, __meta_kubernetes_endpoint_port_name]
//     action: keep
//     regex: default;kubernetes;https
//
// - job_name: 'kubernetes-nodes'
//
//   scheme: https
//
//   tls_config:
//     ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
//   bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token
//
//   kubernetes_sd_configs:
//   - role: node
//
//   relabel_configs:
//   - action: labelmap
//     regex: __meta_kubernetes_node_label_(.+)
//   - target_label: __address__
//     replacement: kubernetes.default.svc:443
//   - source_labels: [__meta_kubernetes_node_name]
//     regex: (.+)
//     target_label: __metrics_path__
//     replacement: /api/v1/nodes/${1}/proxy/metrics
//
//
// - job_name: 'kubernetes-pods'
//
//   kubernetes_sd_configs:
//   - role: pod
//
//   relabel_configs:
//   - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
//     action: keep
//     regex: true
//   - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
//     action: replace
//     target_label: __metrics_path__
//     regex: (.+)
//   - source_labels: [__address__, __meta_kubernetes_pod_annotation_prometheus_io_port]
//     action: replace
//     regex: ([^:]+)(?::\d+)?;(\d+)
//     replacement: $1:$2
//     target_label: __address__
//   - action: labelmap
//     regex: __meta_kubernetes_pod_label_(.+)
//   - source_labels: [__meta_kubernetes_namespace]
//     action: replace
//     target_label: kubernetes_namespace
//   - source_labels: [__meta_kubernetes_pod_name]
//     action: replace
//     target_label: kubernetes_pod_name
//
// - job_name: 'kubernetes-cadvisor'
//
//   scheme: https
//
//   tls_config:
//     ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
//   bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token
//
//   kubernetes_sd_configs:
//   - role: node
//
//   relabel_configs:
//   - action: labelmap
//     regex: __meta_kubernetes_node_label_(.+)
//   - target_label: __address__
//     replacement: kubernetes.default.svc:443
//   - source_labels: [__meta_kubernetes_node_name]
//     regex: (.+)
//     target_label: __metrics_path__
//     replacement: /api/v1/nodes/${1}/proxy/metrics/cadvisor
//
// - job_name: 'kubernetes-service-endpoints'
//
//   kubernetes_sd_configs:
//   - role: endpoints
//
//   relabel_configs:
//   - source_labels: [__meta_kubernetes_service_annotation_prometheus_io_scrape]
//     action: keep
//     regex: true
//   - source_labels: [__meta_kubernetes_service_annotation_prometheus_io_scheme]
//     action: replace
//     target_label: __scheme__
//     regex: (https?)
//   - source_labels: [__meta_kubernetes_service_annotation_prometheus_io_path]
//     action: replace
//     target_label: __metrics_path__
//     regex: (.+)
//   - source_labels: [__address__, __meta_kubernetes_service_annotation_prometheus_io_port]
//     action: replace
//     target_label: __address__
//     regex: ([^:]+)(?::\d+)?;(\d+)
//     replacement: $1:$2
//   - action: labelmap
//     regex: __meta_kubernetes_service_label_(.+)
//   - source_labels: [__meta_kubernetes_namespace]
//     action: replace
//     target_label: kubernetes_namespace
//   - source_labels: [__meta_kubernetes_service_name]
//     action: replace
//     target_label: kubernetes_name`
//
//const alertManagerDefaultTemplate = `{{ define "__alertmanager" }}AlertManager{{ end }}
//{{ define "__alertmanagerURL" }}{{ .ExternalURL }}/#/alerts?receiver={{ .Receiver }}{{ end }}
//{{ define "__subject" }}[{{ .Status | toUpper }}{{ if eq .Status "firing" }}:{{ .Alerts.Firing | len }}{{ end }}] {{ .GroupLabels.SortedPairs.Values | join " " }} {{ if gt (len .CommonLabels) (len .GroupLabels) }}({{ with .CommonLabels.Remove .GroupLabels.Names }}{{ .Values | join " " }}{{ end }}){{ end }}{{ end }}
//{{ define "__description" }}{{ end }}
//{{ define "__text_alert_list" }}{{ range . }}Labels:
//{{ range .Labels.SortedPairs }} - {{ .Name }} = {{ .Value }}
//{{ end }}Annotations:
//{{ range .Annotations.SortedPairs }} - {{ .Name }} = {{ .Value }}
//{{ end }}Source: {{ .GeneratorURL }}
//{{ end }}{{ end }}
//{{ define "slack.default.title" }}{{ template "__subject" . }}{{ end }}
//{{ define "slack.default.username" }}{{ template "__alertmanager" . }}{{ end }}
//{{ define "slack.default.fallback" }}{{ template "slack.default.title" . }} | {{ template "slack.default.titlelink" . }}{{ end }}
//{{ define "slack.default.pretext" }}{{ end }}
//{{ define "slack.default.titlelink" }}{{ template "__alertmanagerURL" . }}{{ end }}
//{{ define "slack.default.iconemoji" }}{{ end }}
//{{ define "slack.default.iconurl" }}{{ end }}
//{{ define "slack.default.text" }}{{ end }}
//{{ define "hipchat.default.from" }}{{ template "__alertmanager" . }}{{ end }}
//{{ define "hipchat.default.message" }}{{ template "__subject" . }}{{ end }}
//{{ define "pagerduty.default.description" }}{{ template "__subject" . }}{{ end }}
//{{ define "pagerduty.default.client" }}{{ template "__alertmanager" . }}{{ end }}
//{{ define "pagerduty.default.clientURL" }}{{ template "__alertmanagerURL" . }}{{ end }}
//{{ define "pagerduty.default.instances" }}{{ template "__text_alert_list" . }}{{ end }}
//{{ define "opsgenie.default.message" }}{{ template "__subject" . }}{{ end }}
//{{ define "opsgenie.default.description" }}{{ .CommonAnnotations.SortedPairs.Values | join " " }}
//{{ if gt (len .Alerts.Firing) 0 -}}
//Alerts Firing:
//{{ template "__text_alert_list" .Alerts.Firing }}
//{{- end }}
//{{ if gt (len .Alerts.Resolved) 0 -}}
//Alerts Resolved:
//{{ template "__text_alert_list" .Alerts.Resolved }}
//{{- end }}
//{{- end }}
//{{ define "opsgenie.default.source" }}{{ template "__alertmanagerURL" . }}{{ end }}
//{{ define "victorops.default.message" }}{{ template "__subject" . }} | {{ template "__alertmanagerURL" . }}{{ end }}
//{{ define "victorops.default.from" }}{{ template "__alertmanager" . }}{{ end }}
//{{ define "email.default.subject" }}{{ template "__subject" . }}{{ end }}
//{{ define "email.default.html" }}
//<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
//<!--
//Style and HTML derived from https://github.com/mailgun/transactional-email-templates
//The MIT License (MIT)
//Copyright (c) 2014 Mailgun
//Permission is hereby granted, free of charge, to any person obtaining a copy
//of this software and associated documentation files (the "Software"), to deal
//in the Software without restriction, including without limitation the rights
//to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
//copies of the Software, and to permit persons to whom the Software is
//furnished to do so, subject to the following conditions:
//The above copyright notice and this permission notice shall be included in all
//copies or substantial portions of the Software.
//THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
//IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
//FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
//AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
//LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
//OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
//SOFTWARE.
//-->
//<html xmlns="http://www.w3.org/1999/xhtml" xmlns="http://www.w3.org/1999/xhtml" style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;">
//<head style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;">
//<meta name="viewport" content="width=device-width" style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;" />
//<meta http-equiv="Content-Type" content="text/html; charset=UTF-8" style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;" />
//<title style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;">{{ template "__subject" . }}</title>
//</head>
//<body itemscope="" itemtype="http://schema.org/EmailMessage" style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; -webkit-font-smoothing: antialiased; -webkit-text-size-adjust: none; height: 100%; line-height: 1.6em; width: 100% !important; background-color: #f6f6f6; margin: 0; padding: 0;" bgcolor="#f6f6f6">
//<table style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; width: 100%; background-color: #f6f6f6; margin: 0;" bgcolor="#f6f6f6">
// <tr style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;">
//   <td style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; vertical-align: top; margin: 0;" valign="top"></td>
//   <td width="600" style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; vertical-align: top; display: block !important; max-width: 600px !important; clear: both !important; width: 100% !important; margin: 0 auto; padding: 0;" valign="top">
//     <div style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; max-width: 600px; display: block; margin: 0 auto; padding: 0;">
//       <table width="100%" cellpadding="0" cellspacing="0" style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; border-radius: 3px; background-color: #fff; margin: 0; border: 1px solid #e9e9e9;" bgcolor="#fff">
//         <tr style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;">
//           <td style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 16px; vertical-align: top; color: #fff; font-weight: 500; text-align: center; border-radius: 3px 3px 0 0; background-color: #E6522C; margin: 0; padding: 20px;" align="center" bgcolor="#E6522C" valign="top">
//             {{ .Alerts | len }} alert{{ if gt (len .Alerts) 1 }}s{{ end }} for {{ range .GroupLabels.SortedPairs }}
//               {{ .Name }}={{ .Value }}
//             {{ end }}
//           </td>
//         </tr>
//         <tr style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;">
//           <td style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; vertical-align: top; margin: 0; padding: 10px;" valign="top">
//             <table width="100%" cellpadding="0" cellspacing="0" style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;">
//               <tr style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;">
//                 <td style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; vertical-align: top; margin: 0; padding: 0 0 20px;" valign="top">
//                   <a href="{{ template "__alertmanagerURL" . }}" style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; color: #FFF; text-decoration: none; line-height: 2em; font-weight: bold; text-align: center; cursor: pointer; display: inline-block; border-radius: 5px; text-transform: capitalize; background-color: #348eda; margin: 0; border-color: #348eda; border-style: solid; border-width: 10px 20px;">View in {{ template "__alertmanager" . }}</a>
//                 </td>
//               </tr>
//               {{ if gt (len .Alerts.Firing) 0 }}
//               <tr style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;">
//                 <td style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; vertical-align: top; margin: 0; padding: 0 0 20px;" valign="top">
//                   <strong style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;">[{{ .Alerts.Firing | len }}] Firing</strong>
//                 </td>
//               </tr>
//               {{ end }}
//               {{ range .Alerts.Firing }}
//               <tr style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;">
//                 <td style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; vertical-align: top; margin: 0; padding: 0 0 20px;" valign="top">
//                   <strong style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;">Labels</strong><br style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;" />
//                   {{ range .Labels.SortedPairs }}{{ .Name }} = {{ .Value }}<br style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;" />{{ end }}
//                   {{ if gt (len .Annotations) 0 }}<strong style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;">Annotations</strong><br style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;" />{{ end }}
//                   {{ range .Annotations.SortedPairs }}{{ .Name }} = {{ .Value }}<br style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;" />{{ end }}
//                   <a href="{{ .GeneratorURL }}" style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; color: #348eda; text-decoration: underline; margin: 0;">Source</a><br style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;" />
//                 </td>
//               </tr>
//               {{ end }}
//               {{ if gt (len .Alerts.Resolved) 0 }}
//                 {{ if gt (len .Alerts.Firing) 0 }}
//               <tr style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;">
//                 <td style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; vertical-align: top; margin: 0; padding: 0 0 20px;" valign="top">
//                   <br style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;" />
//                   <hr style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;" />
//                   <br style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;" />
//                 </td>
//               </tr>
//                 {{ end }}
//               <tr style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;">
//                 <td style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; vertical-align: top; margin: 0; padding: 0 0 20px;" valign="top">
//                   <strong style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;">[{{ .Alerts.Resolved | len }}] Resolved</strong>
//                 </td>
//               </tr>
//               {{ end }}
//               {{ range .Alerts.Resolved }}
//               <tr style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;">
//                 <td style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; vertical-align: top; margin: 0; padding: 0 0 20px;" valign="top">
//                   <strong style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;">Labels</strong><br style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;" />
//                   {{ range .Labels.SortedPairs }}{{ .Name }} = {{ .Value }}<br style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;" />{{ end }}
//                   {{ if gt (len .Annotations) 0 }}<strong style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;">Annotations</strong><br style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;" />{{ end }}
//                   {{ range .Annotations.SortedPairs }}{{ .Name }} = {{ .Value }}<br style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;" />{{ end }}
//                   <a href="{{ .GeneratorURL }}" style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; color: #348eda; text-decoration: underline; margin: 0;">Source</a><br style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;" />
//                 </td>
//               </tr>
//               {{ end }}
//             </table>
//           </td>
//         </tr>
//       </table>
//       <div style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; width: 100%; clear: both; color: #999; margin: 0; padding: 20px;">
//         <table width="100%" style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;">
//           <tr style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; margin: 0;">
//             <td style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 12px; vertical-align: top; text-align: center; color: #999; margin: 0; padding: 0 0 20px;" align="center" valign="top"><a href="{{ .ExternalURL }}" style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 12px; color: #999; text-decoration: underline; margin: 0;">Sent by {{ template "__alertmanager" . }}</a></td>
//           </tr>
//         </table>
//       </div></div>
//   </td>
//   <td style="font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; box-sizing: border-box; font-size: 14px; vertical-align: top; margin: 0;" valign="top"></td>
// </tr>
//</table>
//</body>
//</html>
//{{ end }}
//{{ define "pushover.default.title" }}{{ template "__subject" . }}{{ end }}
//{{ define "pushover.default.message" }}{{ .CommonAnnotations.SortedPairs.Values | join " " }}
//{{ if gt (len .Alerts.Firing) 0 }}
//Alerts Firing:
//{{ template "__text_alert_list" .Alerts.Firing }}
//{{ end }}
//{{ if gt (len .Alerts.Resolved) 0 }}
//Alerts Resolved:
//{{ template "__text_alert_list" .Alerts.Resolved }}
//{{ end }}
//{{ end }}
//{{ define "pushover.default.url" }}{{ template "__alertmanagerURL" . }}{{ end }}`
//
//const alertManagerSlackTemplate = `{{ define "slack.devops.text" }}
//{{range .Alerts}}{{.Annotations.DESCRIPTION}}
//{{end}}
//{{ end }}`
//
//type PrometheusHelper struct {
//	KubeApi *kubernetes.Clientset
//}
//
//// NewPrometheusHelper creates a new PrometheusHelper
//func NewPrometheusHelper() (*PrometheusHelper, error) {
//
//	config, err := rest.InClusterConfig()
//	if err != nil {
//		return nil, err
//	}
//	clientset, err := kubernetes.NewForConfig(config)
//
//	if err != nil {
//		return nil, err
//	}
//
//	return &PrometheusHelper{KubeApi: clientset}, nil
//}
//
//// CreateOrUpdatePrometheusNamespace creates or updates the Prometheus namespace
//func (p *PrometheusHelper) CreateOrUpdatePrometheusNamespace() error {
//	namespace := &v1.Namespace{
//		ObjectMeta: metav1.ObjectMeta{
//			Name: "monitoring",
//		},
//	}
//	_, err := p.KubeApi.CoreV1().Namespaces().Create(namespace)
//
//	if err != nil {
//		_, err = p.KubeApi.CoreV1().Namespaces().Update(namespace)
//		if err != nil {
//			return err
//		}
//	}
//	return nil
//}
//
//// CreateOrUpdatePrometheusConfigMap creates or updates the Prometheus config map
//func (p *PrometheusHelper) CreateOrUpdatePrometheusConfigMap() error {
//	cm := &v1.ConfigMap{
//		ObjectMeta: metav1.ObjectMeta{
//			Name:      "prometheus-server-conf",
//			Namespace: "monitoring",
//			Labels:    map[string]string{},
//		},
//		Data: map[string]string{},
//	}
//	cm.ObjectMeta.Labels["name"] = "prometheus-server-conf"
//
//	var configYaml interface{}
//	err := yaml.Unmarshal([]byte(prometheusYml), &configYaml)
//	if err != nil {
//		return err
//	}
//	yamlString, err := yaml.Marshal(configYaml)
//	if err != nil {
//		return err
//	}
//	cm.Data["prometheus.yml"] = string(yamlString)
//
//	return p.createOrUpdateConfigMap(cm)
//}
//
//// CreateOrUpdatePrometheusConfigMap creates or updates the Prometheus config map
//func (p *PrometheusHelper) CreateOrUpdatePrometheusClusterRole() error {
//	role := &v1beta1.ClusterRole{
//		ObjectMeta: metav1.ObjectMeta{
//			Name: "prometheus",
//		},
//		Rules: []v1beta1.PolicyRule{
//			{
//				Verbs:     []string{"get", "list", "watch"},
//				APIGroups: []string{""},
//				Resources: []string{"nodes", "nodes/proxy", "services", "endpoints", "pods"},
//			},
//			{
//				Verbs:     []string{"get", "list", "watch"},
//				APIGroups: []string{"extensions"},
//				Resources: []string{"ingresses"},
//			},
//			{
//				Verbs:           []string{"get"},
//				NonResourceURLs: []string{"/metrics"},
//			},
//		},
//	}
//	_, err := p.KubeApi.RbacV1beta1().ClusterRoles().Create(role)
//	if err != nil {
//		_, err := p.KubeApi.RbacV1beta1().ClusterRoles().Update(role)
//		if err != nil {
//			return err
//		}
//	}
//
//	binding := &v1beta1.ClusterRoleBinding{
//		ObjectMeta: metav1.ObjectMeta{
//			Name: "prometheus",
//		},
//		Subjects: []v1beta1.Subject{
//			{
//				Kind:      "ServiceAccount",
//				Name:      "default",
//				Namespace: "monitoring",
//			},
//		},
//		RoleRef: v1beta1.RoleRef{
//			APIGroup: "rbac.authorization.k8s.io",
//			Kind:     "ClusterRole",
//			Name:     "prometheus",
//		},
//	}
//	_, err = p.KubeApi.RbacV1beta1().ClusterRoleBindings().Create(binding)
//
//	if err != nil {
//		_, err = p.KubeApi.RbacV1beta1().ClusterRoleBindings().Update(binding)
//		if err != nil {
//			return err
//		}
//	}
//
//	return nil
//}
//
//// CreateOrUpdatePrometheusDeployment creates or updates the Prometheus config map
//func (p *PrometheusHelper) CreateOrUpdatePrometheusDeployment() error {
//	deployment := &appsv1.Deployment{
//		ObjectMeta: metav1.ObjectMeta{
//			Name:      "prometheus-deployment",
//			Namespace: "monitoring",
//		},
//		Spec: appsv1.DeploymentSpec{
//			Replicas: int32Ptr(1),
//			Selector: &metav1.LabelSelector{
//				MatchLabels: map[string]string{
//					"app": "prometheus-server",
//				},
//			},
//			Template: v1.PodTemplateSpec{
//				ObjectMeta: metav1.ObjectMeta{
//					Labels: map[string]string{
//						"app": "prometheus-server",
//					},
//				},
//				Spec: v1.PodSpec{
//					Volumes: []v1.Volume{
//						{
//							Name: "prometheus-config-volume",
//							VolumeSource: v1.VolumeSource{
//								ConfigMap: &v1.ConfigMapVolumeSource{
//									LocalObjectReference: v1.LocalObjectReference{
//										Name: "prometheus-server-conf",
//									},
//									DefaultMode: int32Ptr(420),
//								},
//							},
//						},
//						{
//							Name: "prometheus-storage-volume",
//							VolumeSource: v1.VolumeSource{
//								EmptyDir: &v1.EmptyDirVolumeSource{},
//							},
//						},
//					},
//					Containers: []v1.Container{
//						{
//							Name:  "prometheus",
//							Image: "prom/prometheus:v2.12.0",
//							Args:  []string{"--config.file=/etc/prometheus/prometheus.yml", "--storage.tsdb.path=/prometheus/"},
//							Ports: []v1.ContainerPort{
//								{
//									ContainerPort: 9090,
//								},
//							},
//							VolumeMounts: []v1.VolumeMount{
//								{
//									Name:      "prometheus-config-volume",
//									MountPath: "/etc/prometheus/",
//								},
//								{
//									Name:      "prometheus-storage-volume",
//									MountPath: "/prometheus/",
//								},
//							},
//						},
//					},
//				},
//			},
//		},
//	}
//	err := p.createOrUpdateDeployment(deployment)
//	if err != nil {
//		return err
//	}
//
//	service := &v1.Service{
//		ObjectMeta: metav1.ObjectMeta{
//			Name:      "prometheus-service",
//			Namespace: "monitoring",
//			Annotations: map[string]string{
//				"prometheus.io/scrape": "true",
//				"prometheus.io/path":   "/",
//				"prometheus.io.port":   "8080",
//			},
//		},
//		Spec: v1.ServiceSpec{
//			Ports: []v1.ServicePort{
//				{
//					Port: 8080,
//					TargetPort: intstr.IntOrString{
//						IntVal: 9090,
//						Type:   intstr.Int,
//					},
//					NodePort: 30000,
//				},
//			},
//			Selector: map[string]string{
//				"app": "prometheus-server",
//			},
//			Type: "NodePort",
//		},
//	}
//	return p.createOrUpdateService(service)
//}
//
//func (p *PrometheusHelper) createOrUpdateService(service *v1.Service) error {
//	_, err := p.KubeApi.CoreV1().Services("monitoring").Create(service)
//	if err != nil {
//		_, err = p.KubeApi.CoreV1().Services("monitoring").Update(service)
//		if err != nil {
//			return err
//		}
//	}
//	return nil
//}
//
//func (p *PrometheusHelper) createOrUpdateDeployment(deployment *appsv1.Deployment) error {
//	_, err := p.KubeApi.AppsV1().Deployments("monitoring").Create(deployment)
//	if err != nil {
//		_, err := p.KubeApi.AppsV1().Deployments("monitoring").Update(deployment)
//		if err != nil {
//			return err
//		}
//	}
//	return nil
//}
//
//// CreateOrUpdateAlertManagerConfigMap creates or updates the Prometheus config map
//func (p *PrometheusHelper) CreateOrUpdateAlertManagerConfigMap() error {
//	cm := &v1.ConfigMap{
//		ObjectMeta: metav1.ObjectMeta{
//			Name:      "alertmanager-config",
//			Namespace: "monitoring",
//		},
//		Data: map[string]string{},
//	}
//
//	var configYaml interface{}
//	err := yaml.Unmarshal([]byte(alertManagerYml), &configYaml)
//	if err != nil {
//		return err
//	}
//	yamlString, err := yaml.Marshal(configYaml)
//	if err != nil {
//		return err
//	}
//	cm.Data["config.yml"] = string(yamlString)
//
//	return p.createOrUpdateConfigMap(cm)
//}
//
//func (p *PrometheusHelper) createOrUpdateConfigMap(cm *v1.ConfigMap) error {
//	_, err := p.KubeApi.CoreV1().ConfigMaps("monitoring").Create(cm)
//	if err != nil {
//		_, err := p.KubeApi.CoreV1().ConfigMaps("monitoring").Update(cm)
//		if err != nil {
//			return err
//		}
//	}
//	return nil
//}
//
//// CreateOrUpdateAlertManagerTemplatesConfigMap creates or updates the Prometheus config map
//func (p *PrometheusHelper) CreateOrUpdateAlertManagerTemplatesConfigMap() error {
//	cm := &v1.ConfigMap{
//		ObjectMeta: metav1.ObjectMeta{
//			Name:      "alertmanager-templates",
//			Namespace: "monitoring",
//		},
//		Data: map[string]string{},
//	}
//
//	cm.Data["default.tmpl"] = alertManagerDefaultTemplate
//	cm.Data["slack.tmpl"] = alertManagerSlackTemplate
//
//	return p.createOrUpdateConfigMap(cm)
//}
//
//// CreateOrUpdateAlertManagerDeployment creates or updates the Prometheus config map
//func (p *PrometheusHelper) CreateOrUpdateAlertManagerDeployment() error {
//	deployment := &appsv1.Deployment{
//		ObjectMeta: metav1.ObjectMeta{
//			Name:      "alertmanager",
//			Namespace: "monitoring",
//		},
//		Spec: appsv1.DeploymentSpec{
//			Replicas: int32Ptr(1),
//			Selector: &metav1.LabelSelector{
//				MatchLabels: map[string]string{
//					"app": "alertmanager",
//				},
//			},
//			Template: v1.PodTemplateSpec{
//				ObjectMeta: metav1.ObjectMeta{
//					Labels: map[string]string{
//						"app": "alertmanager",
//					},
//				},
//				Spec: v1.PodSpec{
//					Volumes: []v1.Volume{
//						{
//							Name: "config-volume",
//							VolumeSource: v1.VolumeSource{
//								ConfigMap: &v1.ConfigMapVolumeSource{
//									LocalObjectReference: v1.LocalObjectReference{
//										Name: "alertmanager-config",
//									},
//								},
//							},
//						},
//						{
//							Name: "templates-volume",
//							VolumeSource: v1.VolumeSource{
//								ConfigMap: &v1.ConfigMapVolumeSource{
//									LocalObjectReference: v1.LocalObjectReference{
//										Name: "alertmanager-templates",
//									},
//								},
//							},
//						},
//						{
//							Name: "alertmanager",
//							VolumeSource: v1.VolumeSource{
//								EmptyDir: &v1.EmptyDirVolumeSource{},
//							},
//						},
//					},
//					Containers: []v1.Container{
//						{
//							Name:  "alertmanager",
//							Image: "prom/alertmanager:latest",
//							Args: []string{
//								"--config.file=/etc/alertmanager/config.yml",
//								"--storage.path=/alertmanager",
//							},
//							Ports: []v1.ContainerPort{
//								{
//									Name:          "alertmanager",
//									ContainerPort: 9093,
//								},
//							},
//							VolumeMounts: []v1.VolumeMount{
//								{
//									Name:      "config-volume",
//									MountPath: "/etc/alertmanager",
//								},
//								{
//									Name:      "templates-volume",
//									MountPath: "/etc/alertmanager-templates",
//								},
//								{
//									Name:      "alertmanager",
//									MountPath: "/alertmanager",
//								},
//							},
//						},
//					},
//				},
//			},
//		},
//	}
//	err := p.createOrUpdateDeployment(deployment)
//	if err != nil {
//		return err
//	}
//	return nil
//}
//
//// CreateOrUpdateAlertManagerService creates or updates the Prometheus config map
//func (p *PrometheusHelper) CreateOrUpdateAlertManagerService() error {
//	service := &v1.Service{
//		ObjectMeta: metav1.ObjectMeta{
//			Name:      "alertmanager",
//			Namespace: "monitoring",
//			Annotations: map[string]string{
//				"prometheus.io/scrape": "true",
//				"prometheus.io/path":   "/",
//				"prometheus.io.port":   "8080",
//			},
//		},
//		Spec: v1.ServiceSpec{
//			Ports: []v1.ServicePort{
//				{
//					Port: 9093,
//					TargetPort: intstr.IntOrString{
//						IntVal: 9093,
//						Type:   intstr.Int,
//					},
//					NodePort: 31000,
//				},
//			},
//			Selector: map[string]string{
//				"app": "alertmanager",
//			},
//			Type: "NodePort",
//		},
//	}
//
//	return p.createOrUpdateService(service)
//}
//
//func int32Ptr(i int32) *int32 { return &i }