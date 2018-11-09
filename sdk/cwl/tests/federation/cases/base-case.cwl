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
    dockerPull: arvados/fed-test:base-case
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
    out: [hash]
    run: md5sum.cwl