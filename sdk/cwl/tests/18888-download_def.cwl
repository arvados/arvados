# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

cwlVersion: v1.2
class: CommandLineTool

$namespaces:
  arv: "http://arvados.org/cwl#"

requirements:
  NetworkAccess:
    networkAccess: true
  arv:RuntimeConstraints:
    outputDirType: keep_output_dir

inputs:
  scripts:
    type: Directory
    default:
      class: Directory
      location: scripts/
outputs:
  out:
    type: Directory
    outputBinding:
      glob: "."

arguments: [$(inputs.scripts.path)/download_all_data.sh, "."]
