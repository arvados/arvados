# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

cwlVersion: v1.0
class: Workflow
$namespaces:
  arv: "http://arvados.org/cwl#"
  cwltool: "http://commonwl.org/cwltool#"
inputs:
  i:
    type: File
    secondaryFiles:
      - .fai
outputs: []
steps:
  step1:
    in:
      i: i
    out: []
    run: ls.cwl
