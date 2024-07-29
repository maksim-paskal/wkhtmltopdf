{{- define "wkhtmltopdf.fullname" -}}
{{- default (printf "%s-%s" .Release.Name "wkhtmltopdf" | trunc 63 | trimSuffix "-") .Values.fullNameOverwrite -}}
{{- end -}}