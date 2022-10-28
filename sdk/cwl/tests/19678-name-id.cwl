# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

class: Workflow
cwlVersion: v1.1
inputs:
  - type:
      fields:
        - name: first
          type: string
        - name: last
          type: string
      type: record
    id: name
outputs:
  - type:
      fields:
        - name: first
          type: string
        - name: last
          type: string
      type: record
    id: processed_name
    outputSource: name
steps: []
