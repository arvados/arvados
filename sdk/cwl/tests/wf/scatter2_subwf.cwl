# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

{
  "$graph": [
    {
      "$namespaces": {
        "arv": "http://arvados.org/cwl#"
      },
      "class": "Workflow",
      "cwlVersion": "v1.0",
      "hints": [],
      "id": "#main",
      "inputs": [
        {
          "id": "#main/fileblub",
          "type": "File"
        },
        {
          "id": "#main/sleeptime",
          "type": "int"
        }
      ],
      "outputs": [
        {
          "id": "#main/out",
          "outputSource": "#main/sleep1/out",
          "type": "string"
        }
      ],
      "requirements": [
        {
          "class": "InlineJavascriptRequirement"
        },
        {
          "class": "ScatterFeatureRequirement"
        },
        {
          "class": "StepInputExpressionRequirement"
        },
        {
          "class": "SubworkflowFeatureRequirement"
        }
      ],
      "steps": [
        {
          "id": "#main/sleep1",
          "in": [
            {
              "id": "#main/sleep1/blurb",
              "valueFrom": "${\n  return String(inputs.sleeptime) + \"b\";\n}\n"
            },
            {
              "id": "#main/sleep1/sleeptime",
              "source": "#main/sleeptime"
            }
          ],
          "out": [
            "#main/sleep1/out"
          ],
          "run": {
            "baseCommand": "sleep",
            "class": "CommandLineTool",
            "id": "#main/sleep1/subtool",
            "inputs": [
              {
                "id": "#main/sleep1/subtool/sleeptime",
                "inputBinding": {
                  "position": 1
                },
                "type": "int"
              }
            ],
            "outputs": [
              {
                "id": "#main/sleep1/subtool/out",
                "outputBinding": {
                  "outputEval": "out"
                },
                "type": "string"
              }
            ]
          }
        }
      ]
    }
  ],
  "$namespaces": {
    "arv": "http://arvados.org/cwl#"
  },
  "cwlVersion": "v1.0"
}
