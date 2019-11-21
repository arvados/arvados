cwlVersion: v1.0
class: Workflow
$namespaces:
  arv: "http://arvados.org/cwl#"
  cwltool: "http://commonwl.org/cwltool#"

inputs:
  arvbox_base: Directory
  branch:
    type: string
    default: master
outputs:
  arvados_api_hosts:
    type: string[]
    outputSource: start/arvados_api_hosts
  arvados_cluster_ids:
    type: string[]
    outputSource: start/arvados_cluster_ids
  superuser_tokens:
    type: string[]
    outputSource: start/superuser_tokens
  arvbox_containers:
    type: string[]
    outputSource: start/arvbox_containers
  arvbox_bin:
    type: File
    outputSource: start/arvbox_bin
  refspec:
    type: string
    outputSource: branch
requirements:
  SubworkflowFeatureRequirement: {}
  ScatterFeatureRequirement: {}
  StepInputExpressionRequirement: {}
  cwltool:LoadListingRequirement:
    loadListing: no_listing
steps:
  start:
    in:
      arvbox_base: arvbox_base
      branch: branch
      logincluster:
        default: true
    out: [arvados_api_hosts, arvados_cluster_ids, arvado_api_host_insecure, superuser_tokens, arvbox_containers, arvbox_bin]
    run: ../../../cwl/tests/federation/arvbox-make-federation.cwl
