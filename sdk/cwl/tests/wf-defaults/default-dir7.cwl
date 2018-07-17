# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

cwlVersion: v1.0
class: Workflow
inputs:
  inp2:
    type: Directory
    default:
      class: Directory
      location: inp1
outputs: []
$namespaces:
  arv: "http://arvados.org/cwl#"
steps:
  step1:
    in:
      inp2: inp2
    out: []
    run: default-dir7a.cwl