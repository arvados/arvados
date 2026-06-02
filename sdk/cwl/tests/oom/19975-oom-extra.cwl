# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

cwlVersion: v1.2
class: CommandLineTool
$namespaces:
  arv: "http://arvados.org/cwl#"

hints:
  arv:OutOfMemoryRetry:
    # legacy misspelled name, should behave exactly the same
    memoryRetryMultipler: 2
    memoryErrorRegex: NoMoreRAM
  ResourceRequirement:
    ramMin: 256
  arv:APIRequirement: {}

baseCommand: python3
inputs:
  oom_script:
    type: File
    default:
      class: File
      path: fakeoom.py
    inputBinding:
      position: 0
  fail_with:
    type: string?
    doc: Fail with this exit code (numeric) or message
    inputBinding:
      position: 1
      prefix: "--fail-with"
  fail_under:
    type: int?
    doc: Fail when the container has less than this much RAM (in SI MB)
    inputBinding:
      position: 2
      prefix: "--fail-under"

outputs: []
