{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "cluster-proxy-addon.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "cluster-proxy-addon.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "cluster-proxy-addon.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Define anp route host.
*/}}
{{- define "cluster-proxy-addon.anpPublicHost" -}}
{{- $firstfourchars := .Values.cluster_basedomain | trunc 4 -}}
{{- if eq $firstfourchars "apps" }}
{{- printf "%s.%s" .Values.anp_route.name .Values.cluster_basedomain | trimSuffix "-" -}}
{{- else }}
{{- printf "%s.apps.%s" .Values.anp_route.name .Values.cluster_basedomain | trimSuffix "-" -}}
{{- end }}
{{- end -}}

{{/*
Define anp route host.
*/}}
{{- define "cluster-proxy-addon.anpPublicPort" -}}
{{- print 443 -}}
{{- end -}}

{{/*
Define user route host.
*/}}
{{- define "cluster-proxy-addon.userPublicHost" -}}
{{- $firstfourchars := .Values.cluster_basedomain | trunc 4 -}}
{{- if eq $firstfourchars "apps" }}
{{- printf "%s.%s" .Values.user_route.name .Values.cluster_basedomain | trimSuffix "-" -}}
{{- else }}
{{- printf "%s.apps.%s" .Values.user_route.name .Values.cluster_basedomain | trimSuffix "-" -}}
{{- end }}
{{- end -}}

{{/*
Define the namespace of proxy-entrypoint.
*/}}
{{- define "cluster-proxy-addon.proxy-entrypoint-namespace" -}}
{{- printf "proxy-entrypoint.%s" .Release.Namespace -}}
{{- end -}}
