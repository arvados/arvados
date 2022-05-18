# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

cwlVersion: v1.2
class: CommandLineTool
baseCommand: echo
inputs:
  message:
    type: File
    inputBinding:
      position: 1
    default:
      class: File
      location: keep:d7514270f356df848477718d58308cc4+94/b

outputs: []
