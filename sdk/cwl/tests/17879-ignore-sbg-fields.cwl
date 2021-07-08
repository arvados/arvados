# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

$namespaces:
  sbg: https://www.sevenbridges.com/
class: "Workflow"
cwlVersion: v1.1
label: "check that sbg x/y fields are correctly ignored"
inputs:
  - id: sampleName
    type: string
    label: Sample name
    'sbg:x': -22
    'sbg:y': 33.4296875
outputs:
  - id: outstr
    type: string
    outputSource: step1/outstr
steps:
  step1:
    in:
      sampleName: sampleName
    out: [outstr]
    run:
      class: CommandLineTool
      inputs:
        sampleName: string
      stdout: out.txt
      outputs:
        outstr:
          type: string
          outputBinding:
            glob: out.txt
            loadContents: true
            outputEval: $(self[0].contents)
      arguments: [echo, "-n", "foo", $(inputs.sampleName), "bar"]
