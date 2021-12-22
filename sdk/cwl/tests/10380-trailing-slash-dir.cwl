# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

cwlVersion: v1.2
class: CommandLineTool
inputs: []
outputs:
  stuff:
    type: Directory
    outputBinding:
      glob: './foo/'
requirements:
  ShellCommandRequirement: {}
arguments: [{shellQuote: false, valueFrom: "mkdir -p foo && touch baz.txt && touch foo/bar.txt"}]
