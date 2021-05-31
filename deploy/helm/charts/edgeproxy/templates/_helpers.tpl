{{- define "edgeproxy.name" -}}
{{- "edgeproxy" }}
{{- end }}

{{- define "edgeproxy.namespace" -}}
{{- default "libra" .Values.namespace | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "edgeproxy.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "edgeproxy.labels" -}}
helm.sh/chart: {{ include "edgeproxy.chart" . }}
{{ include "edgeproxy.selectorLabels" . }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{- define "edgeproxy.selectorLabels" -}}
app.kubernetes.io/name: {{ include "edgeproxy.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

