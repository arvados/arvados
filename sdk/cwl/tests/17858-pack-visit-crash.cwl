# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

cwlVersion: v1.1
class: ExpressionTool
inputs:
  file1:
    type: File
    default:
      class: File
      location: keep:f225e6259bdd63bc7240599648dde9f1+97/hg19.fa
outputs:
  val: string
expression: "$({val: inputs.file1.path})"
