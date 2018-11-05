cwlVersion: v1.0
class: Workflow
$namespaces:
  arv: "http://arvados.org/cwl#"
  cwltool: "http://commonwl.org/cwltool#"
requirements:
  ScatterFeatureRequirement: {}
  cwltool:LoadListingRequirement:
    loadListing: no_listing
inputs:
  containers: string[]
  arvbox_base: Directory
outputs:
  cluster_ids:
    type: string[]
    outputSource: start/cluster_id
  container_ips:
    type: string[]
    outputSource: start/container_ip
steps:
  mkdir:
    in:
      containers: containers
      arvbox_base: arvbox_base
    out: [arvbox_data]
    run: arvbox-mkdir.cwl
  start:
    in:
      container_name: containers
      arvbox_data: mkdir/arvbox_data
    out: [cluster_id, container_ip]
    scatter: [container_name, arvbox_data]
    scatterMethod: dotproduct
    run: arvbox-start.cwl
