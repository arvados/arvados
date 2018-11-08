# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

cwlVersion: v1.0
class: Workflow
$namespaces:
  arv: "http://arvados.org/cwl#"
  cwltool: "http://commonwl.org/cwltool#"
requirements:
  cwltool:LoadListingRequirement:
    loadListing: no_listing
  SubworkflowFeatureRequirement: {}
inputs:
  arvbox_base: Directory
  acr: string?
outputs: []
steps:
  run-arvbox:
    in:
      containers:
        default: [fedbox1, fedbox2, fedbox3]
      arvbox_base: arvbox_base
    out: [cluster_ids, container_hosts, test_user_uuid, test_user_token]
    run: arvbox-fed.cwl
  run-main:
    in:
      arvados_api_host_home: {source: run-arvbox/container_hosts, valueFrom: "$(self[0])"}
      arvados_home_id: {source: run-arvbox/cluster_ids, valueFrom: "$(self[0])"}
      arvados_api_token: run-arvbox/test_user_token
      arvado_api_host_insecure: {default: true}
      arvados_api_host_clusterB: {source: run-arvbox/container_hosts, valueFrom: "$(self[1])"}
      arvados_clusterB_id: {source: run-arvbox/cluster_ids, valueFrom: "$(self[1])"}
      arvados_api_host_clusterC: {source: run-arvbox/container_hosts, valueFrom: "$(self[2])"}
      arvados_clusterC_id: {source: run-arvbox/cluster_ids, valueFrom: "$(self[2])"}
      acr: acr
    out: [base-case-out, runner-home-step-remote-out]
    run: main.cwl
