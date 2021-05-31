{{- define "librad.name" -}}
{{- $tag := .Values.image.tag | replace "." "-" }}
{{- printf "librad-%s" $tag | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "librad.namespace" -}}
{{- default "libra" .Values.namespace | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "librad.image" -}}
{{ .Values.image.repository }}:{{ .Values.image.tag }}
{{- end }}

{{- define "librad.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "librad.labels" -}}
helm.sh/chart: {{ include "librad.chart" . }}
{{ include "librad.selectorLabels" . }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{- define "librad.selectorLabels" -}}
app.kubernetes.io/name: {{ "librad" }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Values.image.tag | quote }}
{{- end }}

