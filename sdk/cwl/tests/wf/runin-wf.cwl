# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

class: Workflow
cwlVersion: v1.0
$namespaces:
  arv: "http://arvados.org/cwl#"
inputs:
  sleeptime:
    type: int
    default: 5
  fileblub:
    type: File
    default:
      class: File
      location: keep:d7514270f356df848477718d58308cc4+94/a
      secondaryFiles:
        - class: File
          location: keep:d7514270f356df848477718d58308cc4+94/b
outputs:
  out:
    type: string
    outputSource: substep/out
requirements:
  SubworkflowFeatureRequirement: {}
  ScatterFeatureRequirement: {}
  InlineJavascriptRequirement: {}
  StepInputExpressionRequirement: {}
steps:
  substep:
    in:
      sleeptime: sleeptime
      fileblub: fileblub
    out: [out]
    hints:
      - class: arv:RunInSingleContainer
      - class: DockerRequirement
        dockerPull: arvados/jobs:2.2.2
    run:
      class: Workflow
      id: mysub
      inputs:
        fileblub: File
      outputs:
        out:
          type: string
          outputSource: sleep1/out
      steps:
        sleep1:
          in:
            fileblub: fileblub
          out: [out]
          run:
            class: CommandLineTool
            id: subtool
            inputs:
              fileblub:
                type: File
                inputBinding: {position: 1}
            outputs:
              out:
                type: string
                outputBinding:
                  outputEval: $("out")
            baseCommand: cat
