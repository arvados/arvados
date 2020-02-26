# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

cwlVersion: v1.0
class: Workflow
$namespaces:
  arv: "http://arvados.org/cwl#"
  cwltool: "http://commonwl.org/cwltool#"
requirements:
  cwltool:LoadListingRequirement:
    loadListing: no_listing
inputs:
  d: Directory
steps:
  step1:
    in:
      d: d
    out: [out]
    run: wf/16169-step.cwl
outputs:
  out:
    type: File
    outputSource: step1/out
