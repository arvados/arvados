# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

cwlVersion: v1.2
class: ExpressionTool
inputs:
  file1:
    type: Directory
    loadListing: deep_listing
outputs:
  val: Directory
  val2: File[]
requirements:
  InlineJavascriptRequirement: {}
expression: |
  ${
   var val2 = inputs.file1.listing.filter(function (f) { return f.class == 'File'; } );
   return {val: inputs.file1, val2: val2}
  }
