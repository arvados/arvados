# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

cwlVersion: v1.0
class: Workflow
requirements:
  ResourceRequirement:
    coresMin: 1

inputs: []

outputs: []

steps:
  echo_a:
    run: echo_a.cwl
    in: []
    out: []
  echo_b:
    run: echo_b.cwl
    in: []
    out: []
