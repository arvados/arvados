# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

cwlVersion: v1.0
class: Workflow
$namespaces:
  arv: "http://arvados.org/cwl#"
requirements:
  SubworkflowFeatureRequirement: {}

inputs: []

outputs: []

steps:
  echo-subwf:
    requirements:
      arv:RunInSingleContainer: {}
    run: echo-subwf.cwl
    in: []
    out: []
