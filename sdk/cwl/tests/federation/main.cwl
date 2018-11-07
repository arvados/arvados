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

outputs:
  base-case-out:
    type: Any
    outputSource: base-case/out
  runner-home-step-remote-out:
    type: Any
    outputSource: runner-home-step-remote/out

steps:
  base-case:
    in:
      arvados_api_token: arvados_api_token
      arvado_api_host_insecure: arvado_api_host_insecure
      arvados_api_hosts: arvados_api_hosts
      arvados_cluster_ids: arvados_cluster_ids
      acr: acr
      wf: {default: {class: File, location: md5sum.cwl}}
      obj:
        default:
          inp:
            class: File
            location: whale.txt
        valueFrom: |-
          ${
          self["runOnCluster"] = inputs.arvados_cluster_ids[0];
          return self;
          }
    out: [out]
    run: testcase.cwl

  runner-home-step-remote:
    in:
      arvados_api_token: arvados_api_token
      arvado_api_host_insecure: arvado_api_host_insecure
      arvados_api_hosts: arvados_api_hosts
      arvados_cluster_ids: arvados_cluster_ids
      acr: acr
      wf: {default: {class: File, location: md5sum.cwl}}
      obj:
        default:
          inp:
            class: File
            location: whale.txt
        valueFrom: |-
          ${
          self["runOnCluster"] = inputs.arvados_cluster_ids[1];
          return self;
          }
    out: [out]
    run: testcase.cwl
