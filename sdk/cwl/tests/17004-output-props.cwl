# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

class: Workflow
cwlVersion: v1.2
$namespaces:
  arv: "http://arvados.org/cwl#"
hints:
  arv:OutputCollectionProperties:
    outputProperties:
      foo: bar
      baz: $(inputs.inp.basename)
inputs:
  inp: File
steps:
  cat:
    in:
      inp: inp
    run: cat.cwl
    out: []
outputs: []
