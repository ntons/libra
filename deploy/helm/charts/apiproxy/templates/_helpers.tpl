{{- define "apiproxy.name" -}}
{{- "apiproxy" }}
{{- end }}

{{- define "apiproxy.namespace" -}}
{{- default "libra" .Values.namespace | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "apiproxy.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "apiproxy.labels" -}}
helm.sh/chart: {{ include "apiproxy.chart" . }}
{{ include "apiproxy.selectorLabels" . }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{- define "apiproxy.selectorLabels" -}}
app.kubernetes.io/name: {{ include "apiproxy.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

