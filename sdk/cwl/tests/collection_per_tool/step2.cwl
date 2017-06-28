# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

cwlVersion: v1.0
class: CommandLineTool
inputs:
  c:
    type: File
    default:
      class: File
      location: c.txt
  b:
    type: File
    default:
      class: File
      location: b.txt
outputs: []
arguments: [echo, $(inputs.c), $(inputs.b)]