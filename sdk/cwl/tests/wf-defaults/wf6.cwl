# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

cwlVersion: v1.0
class: Workflow
inputs: []
outputs: []
$namespaces:
  arv: "http://arvados.org/cwl#"
requirements:
  SubworkflowFeatureRequirement: {}
hints:
  DockerRequirement:
    dockerPull: arvados/jobs:1.4.0.20190604172024
steps:
  step1:
    requirements:
      arv:RunInSingleContainer: {}
    in: []
    out: []
    run: default-dir6.cwl
