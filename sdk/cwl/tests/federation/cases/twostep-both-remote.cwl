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
    dockerPull: arvados/fed-test:twostep-both-remote
inputs:
  inp:
    type: File
    inputBinding: {}
  md5sumCluster: string
  revCluster: string
outputs:
  hash:
    type: File
    outputSource: md5sum/hash
steps:
  md5sum:
    in:
      inp: inp
      runOnCluster: md5sumCluster
    out: [hash]
    hints:
      arv:ClusterTarget:
        cluster_id: $(inputs.runOnCluster)
    run: md5sum.cwl
  rev:
    in:
      inp: md5sum/hash
      runOnCluster: revCluster
    out: [revhash]
    hints:
      arv:ClusterTarget:
        cluster_id: $(inputs.runOnCluster)
    run: rev.cwl
