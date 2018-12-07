# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

cwlVersion: v1.0
class: CommandLineTool
$namespaces:
  arv: "http://arvados.org/cwl#"
requirements:
  InlineJavascriptRequirement: {}
inputs:
  inp:
    type: File
outputs:
  revhash:
    type: File
    outputBinding:
      glob: out.txt
stdout: out.txt
arguments: [rev, $(inputs.inp)]
