#!/usr/bin/env cwl-runner
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

class: CommandLineTool
cwlVersion: v1.0
inputs:
  inp1:
    type: File
    default:
      class: File
      location: hello.txt
      secondaryFiles:
        - class: Directory
          location: indir1
outputs: []
baseCommand: 'true'
