# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

class: ExpressionTool
cwlVersion: v1.2
inputs:
  f1: File
  newname: string
outputs:
  out: File
expression: |
  ${
  inputs.f1.basename = inputs.newname;
  return {"out": inputs.f1};
  }
