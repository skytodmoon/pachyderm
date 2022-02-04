{{- /*
SPDX-FileCopyrightText: Pachyderm, Inc. <info@pachyderm.com>
SPDX-License-Identifier: Apache-2.0
*/ -}}
{{- /* vim: set filetype=mustache: */ -}}

{{- define "pachyderm.storageBackend" -}}
{{- if eq .Values.deployTarget "" }}
{{ fail "deployTarget must be set" }}
{{- end }}
{{- if .Values.pachd.storage.backend -}}
{{ .Values.pachd.storage.backend }}
{{- else if eq .Values.deployTarget "AMAZON" -}}
AMAZON
{{- else if eq .Values.deployTarget "GOOGLE" -}}
GOOGLE
{{- else if eq .Values.deployTarget "MICROSOFT" -}}
MICROSOFT
{{- else if eq .Values.deployTarget "LOCAL" -}}
LOCAL
{{- else -}}
{{ fail "pachd.storage.backend required when no matching deploy target found" }}
{{- end -}}
{{- end -}}

{{- define "pachyderm.clusterDeploymentId" -}}
{{ default (randAlphaNum 32) .Values.pachd.clusterDeploymentID }}
{{- end -}}

{{- define "pachyderm.imagePullSecrets" -}}
{{- if .Values.global.imagePullSecrets }}
imagePullSecrets:
  {{- range .Values.global.imagePullSecrets }}
  - name: {{ . }}
  {{- end }}
{{- end }}
{{- end -}}

{{- define "pachyderm.urlProto" -}}
{{- if or .Values.oidc.userAccessibleOauthIssuerHosttls .Values.pachd.externalService.tls.enabled -}}
https
{{- else -}}
http
{{- end -}}
{{- end -}}

{{- define "pachyderm.issuerURI" -}}
{{-  if eq .Values.pachd.service.type "NodePort" -}}
http://pachd:1658
{{- else if  .Values.pachd.externalService.enabled -}}
http://pachd:30658/dex
{{- else -}}
http://pachd:30658
{{- end -}}
{{- end -}}

{{- /*
reactAppRuntimeIssuerURI: The URI without the path of the user accessible issuerURI. 
ie. In local deployments, this is http://localhost:30658, while the issuer URI is http://pachd:30658
In deployments where the issuerURI is user accessible (ie. Via ingress) this would be the issuerURI without the path
*/ -}}
{{- define "pachyderm.reactAppRuntimeIssuerURI" -}}
{{-  if .Values.oidc.userAccessibleOauthIssuerHost -}}
{{- printf "%s://%s" (include "pachyderm.urlProto" .) .Values.oidc.userAccessibleOauthIssuerHost -}}
{{- else  -}}
http://localhost:30658
{{- end }}
{{- end -}}

{{- define "pachyderm.consoleRedirectURI" -}}
{{- if .Values.oidc.userAccessibleOauthIssuerHost -}}
{{- printf "%s://%s/oauth/callback/?inline=true" (include "pachyderm.urlProto" .) .Values.oidc.userAccessibleOauthIssuerHost -}}
{{- else -}}
http://localhost:4000/oauth/callback/?inline=true
{{- end }}
{{- end -}}

{{- define "pachyderm.pachdRedirectURI" -}}
{{-  if .Values.oidc.userAccessibleOauthIssuerHost -}}
{{- printf "%s://%s/authorization-code/callback" (include "pachyderm.urlProto" .) .Values.oidc.userAccessibleOauthIssuerHost -}}
{{- else -}}
http://localhost:30657/authorization-code/callback
{{- end }}
{{- end -}}

{{- define "pachyderm.pachdPeerAddress" -}}
pachd-peer.{{ .Release.Namespace }}.svc.cluster.local:30653
{{- end }}


{{- define "pachyderm.localhostIssuer" -}}
{{- if .Values.pachd.localhostIssuer -}}
  {{- if eq .Values.pachd.localhostIssuer "true" -}}
    true
  {{- else if eq .Values.pachd.localhostIssuer "false" -}}
    false
  {{- else -}}
    {{- fail "pachd.localhostIssuer must either be set to the string value of \"true\" or \"false\"" }}
  {{- end -}}
{{- else if .Values.pachd.activateEnterpriseMember -}}
false
{{- else if not .Values.pachd.externalService.enabled -}}
true
{{- else if .Values.oidc.userAccessibleOauthIssuerHost -}}
false
{{- end -}}
{{- end }}

{{- define "pachyderm.userAccessibleOauthIssuerHost" -}}
{{- if .Values.oidc.userAccessibleOauthIssuerHost -}}
{{ .Values.oidc.userAccessibleOauthIssuerHost }}
{{- else -}}
localhost:30658
{{- end -}}
{{- end -}}

{{- define "pachyderm.idps" -}}
{{- if .Values.oidc.upstreamIDPs }}
{{ toYaml .Values.oidc.upstreamIDPs | indent 4 }}
{{- else if .Values.oidc.mockIDP }}
    - id: test
      name: test
      type: mockPassword
      jsonConfig: '{"username": "admin", "password": "password"}'
{{- else }}
    {{- fail "either oidc.upstreamIDPs or oidc.mockIDP must be set in non-LOCAL deployments" }}
{{- end }}
{{- end }}

{{- define "pachyderm.pachctlurl" -}}
{{- if regexMatch "^[a-z]*:[0-9]*$" .Values.oidc.userAccessibleOauthIssuerHost -}}
{{ .Values.oidc.userAccessibleOauthIssuerHost }}
{{- else if or .Values.oidc.userAccessibleOauthIssuerHosttls .Values.pachd.externalService.tls.enabled -}}
{{- .Values.oidc.userAccessibleOauthIssuerHost }}:443
{{- else }}
{{- .Values.oidc.userAccessibleOauthIssuerHost }}:80
{{- end }}
{{- end }}
