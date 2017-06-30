# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

class: CommandLineTool
cwlVersion: v1.0
$namespaces:
  cwltool: http://commonwl.org/cwltool#
requirements:
  cwltool:LoadListingRequirement:
    loadListing: no_listing
  InlineJavascriptRequirement: {}
inputs:
  d: Directory
outputs:
  out: stdout
stdout: output.txt
arguments:
  [echo, "${if(inputs.d.listing === undefined) {return 'true';} else {return 'false';}}"]