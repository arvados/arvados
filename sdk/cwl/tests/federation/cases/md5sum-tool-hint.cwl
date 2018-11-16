# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

cwlVersion: v1.0
class: CommandLineTool
$namespaces:
  arv: "http://arvados.org/cwl#"
requirements:
  InlineJavascriptRequirement: {}
hints:
  arv:ClusterTarget:
    cluster_id: $(inputs.runOnCluster)
inputs:
  inp: File
  runOnCluster: string
outputs:
  hash:
    type: File
    outputBinding:
      glob: out.txt
stdin: $(inputs.inp.path)
stdout: out.txt
arguments: ["md5sum", "-"]
