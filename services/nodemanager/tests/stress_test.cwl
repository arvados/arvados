# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0
#
#
# Usage: arvados-cwl-runner stress_test.cwl
#
# Submits 100 jobs or containers, creating load on node manager and
# scheduler.

class: Workflow
cwlVersion: v1.0
requirements:
  ScatterFeatureRequirement: {}
  InlineJavascriptRequirement: {}
inputs: []
outputs: []
steps:
  step1:
    in: []
    out: [out]
    run:
      class: ExpressionTool
      inputs: []
      outputs:
        out: int[]
      expression: |
        ${
          var r = [];
          for (var i = 1; i <= 100; i++) {
            r.push(i);
          }
          return {out: r};
        }
  step2:
    in:
      num: step1/out
    out: []
    scatter: num
    run:
      class: CommandLineTool
      requirements:
        ShellCommandRequirement: {}
      inputs:
        num: int
      outputs: []
      arguments: [echo, "starting",
        {shellQuote: false, valueFrom: "&&"},
        sleep, $((101-inputs.num)*2),
        {shellQuote: false, valueFrom: "&&"},
        echo, "the number of the day is", $(inputs.num)]
