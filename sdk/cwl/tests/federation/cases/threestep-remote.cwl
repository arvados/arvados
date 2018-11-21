# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

cwlVersion: v1.0
class: Workflow
$namespaces:
  arv: "http://arvados.org/cwl#"
requirements:
  InlineJavascriptRequirement: {}
  DockerRequirement:
    dockerPull: arvados/fed-test:threestep-remote
  ScatterFeatureRequirement: {}
inputs:
  inp: File
  clusterA: string
  clusterB: string
  clusterC: string
outputs:
  revhash:
    type: File
    outputSource: revC/revhash
steps:
  md5sum:
    in:
      inp: inp
      runOnCluster: clusterA
    out: [hash]
    hints:
      arv:ClusterTarget:
        cluster_id: $(inputs.runOnCluster)
    run: md5sum.cwl
  revB:
    in:
      inp: md5sum/hash
      runOnCluster: clusterB
    out: [revhash]
    hints:
      arv:ClusterTarget:
        cluster_id: $(inputs.runOnCluster)
    run: rev-input-to-output.cwl
  revC:
    in:
      inp: revB/revhash
      runOnCluster: clusterC
    out: [revhash]
    hints:
      arv:ClusterTarget:
        cluster_id: $(inputs.runOnCluster)
    run: rev-input-to-output.cwl