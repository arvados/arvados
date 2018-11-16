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
    dockerPull: arvados/fed-test:runner-home-step-remote
inputs:
  inp:
    type: File
    inputBinding: {}
  runOnCluster: string
outputs:
  hash:
    type: File
    outputSource: md5sum/hash
steps:
  md5sum:
    in:
      inp: inp
      runOnCluster: runOnCluster
    hints:
      arv:ClusterTarget:
        cluster_id: $(inputs.runOnCluster)
    out: [hash]
    run: md5sum.cwl