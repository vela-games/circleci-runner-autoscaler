{{/*
Expand the name of the chart.
*/}}
{{- define "vela-allocator.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "vela-allocator.fullname" -}}
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
{{- define "vela-allocator.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "vela-allocator.labels" -}}
{{ include "vela-allocator.selectorLabels" . }}
helm.sh/chart: {{ include "vela-allocator.chart" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "vela-allocator.selectorLabels" -}}
app.kubernetes.io/name: {{ include "vela-allocator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "vela-allocator.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "vela-allocator.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{- define "vela-allocator.image" -}}
{{ .Values.image.awsAccountId }}.dkr.ecr.{{ .Values.image.region }}.amazonaws.com/{{ .Values.image.prefix }}:{{ .Values.image.tag }}
{{- end }}

{{- define "vela-allocator.secretStoreVolume" -}}
{{- if .Values.secrets.enabled }}
- name: {{ include "vela-allocator.fullname" . }}-secrets
  csi:
    driver: 'secrets-store.csi.k8s.io'
    readOnly: true
    volumeAttributes:
      secretProviderClass: '{{ include "vela-allocator.fullname" . }}-secrets'
{{- end }}
{{- end }}

{{- define "vela-allocator.secretStoreVolumeMount" -}}
{{- if .Values.secrets.enabled }}
- name: {{ include "vela-allocator.fullname" . }}-secrets
  mountPath: /mnt/secrets
  readOnly: true
{{- end }}
{{- end }}

{{- define "vela-allocator.secretStoreObjects" -}}
{{- if .Values.secrets.enabled }}
{{- range $i, $object := .Values.secrets.objects }}
- objectName: "{{ $object.key }}"
  secretPath: "{{ $object.secretPath }}"
  secretKey: "{{ $object.key }}"
{{- end }}
{{- end }}
{{- end }}

{{- define "vela-allocator.secretStoreSecretData" -}}
{{- if .Values.secrets.enabled }}
{{- range $i, $object := .Values.secrets.objects }}
- objectName: "{{ $object.key }}"
  key: "{{ $object.key }}"
{{- end }}
{{- end }}
{{- end }}

{{- define "vela-allocator.secretStoreSecretEnv" -}}
{{- if .Values.secrets.enabled }}
{{- range $i, $object := .Values.secrets.objects }}
- name: {{ $object.envName }}
  valueFrom:
    secretKeyRef:
      name: '{{ include "vela-allocator.fullname" $ }}-secrets'
      key: {{ $object.key }}
{{- end }}
{{- end }}
{{- end }}

{{- define "vela-allocator.volumes" -}}
{{- if or (eq .Values.secrets.enabled true) (or (eq .Values.certificates.tls.enabled true) (eq .Values.certificates.client.enabled true)) }}
volumes:
{{- end }}
{{- if .Values.secrets.enabled }}
  {{- include "vela-allocator.secretStoreVolume" . }}
{{- end }}
{{- if .Values.certificates.tls.enabled }}
- name: tls
  secret:
    secretName: {{ .Values.certificates.tls.secretName }}
{{- end }}
{{- if .Values.certificates.client.enabled }}
- name: client-ca
  secret:
    secretName: {{ .Values.certificates.client.secretName }}
{{- end }}
{{- end }}

{{- define "vela-allocator.volumeMounts" -}}
{{- if or (eq .Values.secrets.enabled true) (or (eq .Values.certificates.tls.enabled true) (eq .Values.certificates.client.enabled true)) }}
volumeMounts:
{{- end }}
{{- if .Values.secrets.enabled }}
  {{- include "vela-allocator.secretStoreVolumeMount" . }}
{{- end }}
{{- if .Values.certificates.tls.enabled }}
- mountPath: /home/allocator/tls
  name: tls
  readOnly: true
{{- end }}
{{- if .Values.certificates.client.enabled }}
- mountPath: /home/allocator/client-ca
  name: client-ca
  readOnly: true
{{- end }}
{{- end }}

{{- define "vela-allocator.grpcPort" -}}
- name: grpc
{{- range $i, $env := .Values.environmentVariables }}
{{- if eq $env.name "ALLOCATOR_PORT" }}
  containerPort: {{ $env.value }}
{{- end }}
{{- end }}
{{- end }}