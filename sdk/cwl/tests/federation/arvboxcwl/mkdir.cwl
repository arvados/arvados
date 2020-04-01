# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

cwlVersion: v1.1
class: CommandLineTool
$namespaces:
  arv: "http://arvados.org/cwl#"
  cwltool: "http://commonwl.org/cwltool#"
inputs:
  containers:
    type:
      type: array
      items: string
      inputBinding:
        position: 3
        valueFrom: |
          ${
          return "base/"+self;
          }
  arvbox_base: Directory
outputs:
  arvbox_data:
    type: Directory[]
    outputBinding:
      glob: |
        ${
        var r = [];
        for (var i = 0; i < inputs.containers.length; i++) {
          r.push("base/"+inputs.containers[i]);
        }
        return r;
        }
requirements:
  InitialWorkDirRequirement:
    listing:
      - entry: $(inputs.arvbox_base)
        entryname: base
        writable: true
  LoadListingRequirement:
    loadListing: no_listing
  InlineJavascriptRequirement: {}
  InplaceUpdateRequirement:
    inplaceUpdate: true
arguments:
  - mkdir
  - "-p"
