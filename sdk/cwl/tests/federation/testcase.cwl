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
  run-acr:
    in:
      arv_token: arvados_api_token
      arv_insecure: arvado_api_host_insecure
      arv_host: {source: arvados_api_hosts, valueFrom: "$(self[0])"}
      acr: acr
      wf: wf
      obj: obj
    out: [out]
    run: run-acr.cwl
