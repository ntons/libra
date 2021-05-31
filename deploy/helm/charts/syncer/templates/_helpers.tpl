{{- define "librad.name" -}}
{{- $tag := .Values.image.tag | replace "." "-" }}
{{- printf "librad-%s" $tag | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "syncer.name" -}}
{{- "syncer" }}
{{- end }}

{{- define "syncer.namespace" -}}
{{- default "libra" .Values.namespace | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "syncer.image" -}}
{{ .Values.image.repository }}:{{ .Values.image.tag }}
{{- end }}

{{- define "syncer.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "syncer.labels" -}}
helm.sh/chart: {{ include "syncer.chart" . }}
{{ include "syncer.selectorLabels" . }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{- define "syncer.selectorLabels" -}}
app.kubernetes.io/name: {{ "syncer" }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Values.image.tag | quote }}
{{- end }}

