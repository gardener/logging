{{/*
Expand the name of the chart.
*/}}
{{- define "fluent-bit-plugin.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Expand the name for plutono resources (ServiceAccount, Service, Deployment).
*/}}
{{- define "fluent-bit-plugin.plutonoName" -}}
{{- printf "%s-plutono" (include "fluent-bit-plugin.name" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Expand the name for prometheus resources.
*/}}
{{- define "fluent-bit-plugin.prometheusName" -}}
{{- printf "%s-prometheus" (include "fluent-bit-plugin.name" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Expand the name for victorialogs resources.
*/}}
{{- define "fluent-bit-plugin.victorialogsName" -}}
{{- printf "%s-victorialogs" (include "fluent-bit-plugin.name" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Expand the name for otel-collector resources.
*/}}
{{- define "fluent-bit-plugin.otelCollectorName" -}}
{{- printf "%s-otel-collector" (include "fluent-bit-plugin.name" .) | trunc 63 | trimSuffix "-" }}
{{- end }}


{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "fluent-bit-plugin.fullname" -}}
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
{{- define "fluent-bit-plugin.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "fluent-bit-plugin.labels" -}}
helm.sh/chart: {{ include "fluent-bit-plugin.chart" . }}
{{ include "fluent-bit-plugin.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "fluent-bit-plugin.selectorLabels" -}}
app.kubernetes.io/name: {{ include "fluent-bit-plugin.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Plutono Common labels
*/}}
{{- define "fluent-bit-plugin.plutonoLabels" -}}
helm.sh/chart: {{ include "fluent-bit-plugin.chart" . }}
{{ include "fluent-bit-plugin.plutonoSelectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}


{{/*
Plutono Selector labels
*/}}
{{- define "fluent-bit-plugin.plutonoSelectorLabels" -}}
app.kubernetes.io/name: {{ include "fluent-bit-plugin.plutonoName" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Prometheus Common labels
*/}}
{{- define "fluent-bit-plugin.prometheusLabels" -}}
helm.sh/chart: {{ include "fluent-bit-plugin.chart" . }}
{{ include "fluent-bit-plugin.prometheusSelectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Prometheus Selector labels
*/}}
{{- define "fluent-bit-plugin.prometheusSelectorLabels" -}}
app.kubernetes.io/name: {{ include "fluent-bit-plugin.prometheusName" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
VictoriaLogs Common labels
*/}}
{{- define "fluent-bit-plugin.victorialogsLabels" -}}
helm.sh/chart: {{ include "fluent-bit-plugin.chart" . }}
{{ include "fluent-bit-plugin.victorialogsSelectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
VictoriaLogs Selector labels
*/}}
{{- define "fluent-bit-plugin.victorialogsSelectorLabels" -}}
app.kubernetes.io/name: {{ include "fluent-bit-plugin.victorialogsName" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
OtelCollector Common labels
*/}}
{{- define "fluent-bit-plugin.otelCollectorLabels" -}}
helm.sh/chart: {{ include "fluent-bit-plugin.chart" . }}
{{ include "fluent-bit-plugin.otelCollectorSelectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
OtelCollector Selector labels
*/}}
{{- define "fluent-bit-plugin.otelCollectorSelectorLabels" -}}
app.kubernetes.io/name: {{ include "fluent-bit-plugin.otelCollectorName" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "fluent-bit-plugin.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "fluent-bit-plugin.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}


{{/*
Create a template for image repository with concatenation of repository and name parameter.
*/}}
{{- define "fluent-bit-plugin.image.repository" -}}
{{- $name := .name -}}
{{- printf "%s/%s:%s" .repository $name .tag -}}
{{- end }}
