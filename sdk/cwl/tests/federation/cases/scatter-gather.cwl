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
    dockerPull: arvados/fed-test:scatter-gather
  ScatterFeatureRequirement: {}
inputs:
  shards: File[]
  clusters: string[]
outputs:
  joined:
    type: File
    outputSource: cat/joined
steps:
  md5sum:
    in:
      inp: shards
      runOnCluster: clusters
    scatter: [inp, runOnCluster]
    scatterMethod: dotproduct
    out: [hash]
    run: md5sum.cwl
  cat:
    in:
      inp: md5sum/hash
    out: [joined]
    run: cat.cwl
