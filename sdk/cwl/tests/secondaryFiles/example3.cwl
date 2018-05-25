# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

class: CommandLineTool
cwlVersion: v1.0
inputs:
  step_input:
    type: File
    secondaryFiles:
      - .idx
    default:
      class: File
      location: hello.txt
outputs: []
baseCommand: echo
