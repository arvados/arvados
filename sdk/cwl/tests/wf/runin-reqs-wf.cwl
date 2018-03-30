# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

class: Workflow
cwlVersion: v1.0
$namespaces:
  arv: "http://arvados.org/cwl#"
inputs:
  sleeptime:
    type: int[]
    default: [1, 2, 3, 4]
outputs:
  out: []
requirements:
  SubworkflowFeatureRequirement: {}
  ScatterFeatureRequirement: {}
  InlineJavascriptRequirement: {}
  StepInputExpressionRequirement: {}
steps:
  substep:
    in:
      sleeptime: sleeptime
    out: []
    hints:
      - class: arv:RunInSingleContainer
      - class: ResourceRequirement
        ramMin: $(inputs.sleeptime*4)
    scatter: sleeptime
    run:
      class: Workflow
      id: mysub
      inputs:
        sleeptime: int
      outputs: []
      steps:
        sleep1:
          in:
            sleeptime: sleeptime
          out: []
          run:
            class: CommandLineTool
            id: subtool
            inputs:
              sleeptime:
                type: int
            outputs: []
            baseCommand: [cat, /proc/meminfo]
