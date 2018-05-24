# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

{
   "cwlVersion": "v1.0",
      "arguments": [
        "true"
      ],
      "class": "CommandLineTool",
      "inputs": [],
      "outputs": [
        {
          "id": "out",
          "outputBinding": {
            "glob": "*.txt"
          },
          "type": [
            "null",
            "File"
          ]
        }
      ]
}