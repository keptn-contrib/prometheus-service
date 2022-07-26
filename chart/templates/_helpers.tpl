{{/*
Expand the name of the chart.
*/}}
{{- define "prometheus-service.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "prometheus-service.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "prometheus-service.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "prometheus-service.labels" -}}
helm.sh/chart: {{ include "prometheus-service.chart" . }}
{{ include "prometheus-service.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "prometheus-service.selectorLabels" -}}
app.kubernetes.io/name: {{ include "prometheus-service.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "prometheus-service.serviceAccountName" -}}
keptn-prometheus-service
{{- end }}


{{/*
Helper functions for auto detecting Prometheus namespace
*/}}
{{- define "prometheus-service.namespace" -}}
    {{- /* Check if autodetect is set */ -}}
    {{- if and (.Values.prometheus.autodetect) (eq .Values.prometheus.namespace "") }}
        {{- $detectedPrometheusServer := list }}

        {{- /* Find prometheus-server service */ -}}
        {{- $services := lookup "v1" "Service" "" "" }}
        {{- range $index, $srv := $services.items }}
            {{- if (eq "prometheus-server" $srv.metadata.name ) }}
                {{- $detectedPrometheusServer = append $detectedPrometheusServer $srv }}
            {{- end }}
        {{- end }}

        {{- if eq (len $detectedPrometheusServer) 0 }}
            {{- fail (printf "Unable to detect Prometheus in the kubernetes cluster! %+v" $services) }}
        {{- end }}
        {{- if gt (len $detectedPrometheusServer) 1 }}
            {{- fail (printf "Detected more than one Prometheus installation: %+v" $detectedPrometheusServer) }}
        {{- end }}

        {{- (index $detectedPrometheusServer 0).metadata.namespace }}
    {{- else }}
        {{- .Values.prometheus.namespace }}
    {{- end }}
{{- end }}

{{/*
Helper functions for auto detecting Prometheus alertmanager namespace
*/}}
{{- define "prometheus-am-service.namespace" -}}
    {{- /* Check if autodetect is set */ -}}
    {{- if and (.Values.prometheus.autodetect_am) (eq .Values.prometheus.namespace_am "") }}
        {{- $detectedPrometheusServer := list }}

        {{- /* Find prometheus-alertmanager service */ -}}
        {{- $services := lookup "v1" "Service" "" "" }}
        {{- range $index, $srv := $services.items }}
            {{- if (eq "prometheus-alertmanager" $srv.metadata.name ) }}
                {{- $detectedPrometheusServer = append $detectedPrometheusServer $srv }}
            {{- end }}
        {{- end }}

        {{- if eq (len $detectedPrometheusServer) 0 }}
            {{- fail (printf "Unable to detect Prometheus in the kubernetes cluster! %+v" $services) }}
        {{- end }}
        {{- if gt (len $detectedPrometheusServer) 1 }}
            {{- fail (printf "Detected more than one Prometheus installation: %+v" $detectedPrometheusServer) }}
        {{- end }}

        {{- (index $detectedPrometheusServer 0).metadata.namespace }}
    {{- else }}
        {{- .Values.prometheus.namespace_am }}
    {{- end }}
{{- end }}
