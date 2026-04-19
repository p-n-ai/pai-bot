{{/*
SPDX-License-Identifier: Apache-2.0
Helm template helpers for pai-bot.
*/}}

{{- define "pai.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "pai.fullname" -}}
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

{{- define "pai.labels" -}}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{ include "pai.selectorLabels" . }}
{{- end }}

{{- define "pai.selectorLabels" -}}
app.kubernetes.io/name: {{ include "pai.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "pai.databaseUrl" -}}
{{- if .Values.postgres.enabled }}
{{- printf "postgres://%s:%s@%s-postgres:5432/%s?sslmode=disable" .Values.postgres.auth.username .Values.postgres.auth.password (include "pai.fullname" .) .Values.postgres.auth.database }}
{{- end }}
{{- end }}

{{- define "pai.cacheUrl" -}}
{{- if .Values.dragonfly.enabled }}
{{- printf "redis://%s-dragonfly:6379" (include "pai.fullname" .) }}
{{- end }}
{{- end }}

{{- define "pai.natsUrl" -}}
{{- if .Values.nats.enabled }}
{{- printf "nats://%s-nats:4222" (include "pai.fullname" .) }}
{{- end }}
{{- end }}
