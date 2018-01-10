# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

class: Workflow
cwlVersion: v1.0
$namespaces:
  arv: "http://arvados.org/cwl#"
inputs:
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
hints:
  arv:IntermediateOutput:
    outputTTL: 60
steps:
  substep:
    in:
      fileblub: fileblub
    out: [out]
    hints:
      - class: arv:RunInSingleContainer
    run:
      class: Workflow
      id: mysub
      inputs:
        fileblub: File
      outputs:
        out:
          type: string
          outputSource: cat1/out
      steps:
        cat1:
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
                  outputEval: "out"
            baseCommand: cat
