#!/usr/bin/env cwl-runner
cwlVersion: v1.0
class: Workflow
$namespaces:
  arv: "http://arvados.org/cwl#"
  cwltool: "http://commonwl.org/cwltool#"
hints:
  cwltool:Secrets:
    secrets: [arvados_api_token]
inputs:
  arvados_api_host_home: string
  arvados_home_id: string
  arvados_api_token: string
  arvado_api_host_insecure:
    type: bool
    default: false
  arvados_api_host_clusterB: string
  arvados_clusterB_id: string
  arvados_api_host_clusterC: string
  arvados_clusterC_id: string

outputs: []
