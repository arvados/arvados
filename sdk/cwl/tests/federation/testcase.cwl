#!/usr/bin/env cwl-runner
cwlVersion: v1.0
class: Workflow
$namespaces:
  arv: "http://arvados.org/cwl#"
  cwltool: "http://commonwl.org/cwltool#"
hints:
  cwltool:Secrets:
    secrets: [arvados_api_token]
requirements:
  StepInputExpressionRequirement: {}
  InlineJavascriptRequirement: {}
  SubworkflowFeatureRequirement: {}
inputs:
  arvados_api_token: string
  arvado_api_host_insecure:
    type: boolean
    default: false
  arvados_api_hosts: string[]
  arvados_cluster_ids: string[]
  acr: string?
  wf: File
  obj: Any
outputs:
  out:
    type: Any
    outputSource: run-acr/out
steps:
  prepare:
    in:
      arvados_api_token: arvados_api_token
      arvado_api_host_insecure: arvado_api_host_insecure
      arvados_api_host: {source: arvados_api_hosts, valueFrom: "$(self[0])"}
      arvados_cluster_ids: arvados_cluster_ids
      wf: wf
      obj: obj
    out: [done]
    run: prepare.cwl
  run-acr:
    in:
      prepare: prepare/done
      arvados_api_token: arvados_api_token
      arvado_api_host_insecure: arvado_api_host_insecure
      arvados_api_host: {source: arvados_api_hosts, valueFrom: "$(self[0])"}
      acr: acr
      wf: wf
      obj: obj
    out: [out]
    run: run-acr.cwl
