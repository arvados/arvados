# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

cwlVersion: v1.2
class: CommandLineTool
$namespaces:
  arv: "http://arvados.org/cwl#"
hints:
  arv:OutOfMemoryRetry:
    memoryRetryMultipler: 2
    memoryErrorRegex: Whoops
  ResourceRequirement:
    ramMin: 256
  arv:APIRequirement: {}
inputs:
  fakeoom: File
outputs: []
arguments: [python3, $(inputs.fakeoom)]
