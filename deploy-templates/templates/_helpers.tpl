{{/*
Expand the name of the chart.
*/}}
{{- define "jenkins-operator.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "jenkins-operator.fullname" -}}
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
{{- define "jenkins-operator.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "jenkins-operator.labels" -}}
helm.sh/chart: {{ include "jenkins-operator.chart" . }}
{{ include "jenkins-operator.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "jenkins-operator.selectorLabels" -}}
app.kubernetes.io/name: {{ include "jenkins-operator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "jenkins-operator.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "jenkins-operator.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Define Jenkins URL
*/}}
{{- define "jenkins-operator.jenkinsBaseUrl" -}}
{{- if .Values.jenkins.basePath }}
{{- .Values.global.dnsWildCard }}
{{- else }}
{{- printf "jenkins-%s.%s" .Values.global.edpName .Values.global.dnsWildCard  }}
{{- end }}
{{- end }}

{{/*
Define Jenkins BasePath
*/}}
{{- define "jenkins-operator.jenkinsBasePath" -}}
{{- if .Values.jenkins.basePath }}
{{- printf "/%s(/|$)(.*)" .Values.jenkins.basePath }}
{{- else }}
{{- printf "/"  }}
{{- end }}
{{- end }}
