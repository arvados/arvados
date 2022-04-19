# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

class: CommandLineTool
cwlVersion: v1.2
inputs:
  p: File
  checkname: string
outputs: []
arguments:
  - sh
  - "-c"
  - |
    name=`basename $(inputs.p.path)`
    ls -l $(inputs.p.path)
    if test $name = $(inputs.checkname) ; then
      echo success
    else
      echo expected basename to be $(inputs.checkname) but was $name
      exit 1
    fi
