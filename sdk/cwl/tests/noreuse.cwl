# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

cwlVersion: v1.0
class: Workflow
$namespaces:
  arv: "http://arvados.org/cwl#"
inputs: []
outputs: []
steps:
  step1:
    in:
      message:
        default: "hello world"
    out: [output]
    hints:
      arv:ReuseRequirement:
        enableReuse: false
    run: stdout.cwl