# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

cwlVersion: v1.0
class: CommandLineTool
$namespaces:
  arv: "http://arvados.org/cwl#"
  cwltool: "http://commonwl.org/cwltool#"
inputs:
  container_name: string
  this_cluster_id: string
  cluster_ids: string[]
  cluster_hosts: string[]
  arvbox_data: Directory
  arvbox_bin: File
outputs:
  arvbox_data_out:
    type: Directory
    outputBinding:
      outputEval: $(inputs.arvbox_data)
requirements:
  EnvVarRequirement:
    envDef:
      ARVBOX_CONTAINER: $(inputs.container_name)
      ARVBOX_DATA: $(inputs.arvbox_data.path)
  InitialWorkDirRequirement:
    listing:
      - entryname: cluster_config.yml.override
        entry: >-
          ${
          var remoteClusters = {};
          for (var i = 0; i < inputs.cluster_ids.length; i++) {
            remoteClusters[inputs.cluster_ids[i]] = {
              "Host": inputs.cluster_hosts[i],
              "Proxy": true,
              "Insecure": true
            };
          }
          var r = {"Clusters": {}};
          r["Clusters"][inputs.this_cluster_id] = {"RemoteClusters": remoteClusters};
          return JSON.stringify(r);
          }
      - entryname: application.yml.override
        entry: >-
          ${
          var remoteClusters = {};
          for (var i = 0; i < inputs.cluster_ids.length; i++) {
            remoteClusters[inputs.cluster_ids[i]] = inputs.cluster_hosts[i];
          }
          return JSON.stringify({"development": {"remote_hosts": remoteClusters}});
          }
  cwltool:LoadListingRequirement:
    loadListing: no_listing
  ShellCommandRequirement: {}
  InlineJavascriptRequirement: {}
  cwltool:InplaceUpdateRequirement:
    inplaceUpdate: true
arguments:
  - shellQuote: false
    valueFrom: |
      docker cp cluster_config.yml.override $(inputs.container_name):/var/lib/arvados
      docker cp application.yml.override $(inputs.container_name):/usr/src/arvados/services/api/config
      $(inputs.arvbox_bin.path) sv restart api
      $(inputs.arvbox_bin.path) sv restart controller
      $(inputs.arvbox_bin.path) sv restart keepstore0
      $(inputs.arvbox_bin.path) sv restart keepstore1
