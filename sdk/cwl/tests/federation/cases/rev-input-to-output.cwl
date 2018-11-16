# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

cwlVersion: v1.0
class: CommandLineTool
$namespaces:
  arv: "http://arvados.org/cwl#"
requirements:
  InlineJavascriptRequirement: {}
  ShellCommandRequirement: {}
inputs:
  inp:
    type: File
outputs:
  original:
    type: File
    outputBinding:
      glob: $(inputs.inp.basename)
  revhash:
    type: stdout
stdout: rev-$(inputs.inp.basename)
arguments:
  - shellQuote: false
    valueFrom: |
      ln -s $(inputs.inp.path) $(inputs.inp.basename) &&
      rev $(inputs.inp.basename)
