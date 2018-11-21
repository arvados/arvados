# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

cwlVersion: v1.0
class: CommandLineTool
requirements:
  InitialWorkDirRequirement:
    listing:
      - entryname: config.json
        entry: |-
          ${
          return JSON.stringify({
            check_collections: inputs.check_collections
          });
          }
  EnvVarRequirement:
    envDef:
      ARVADOS_API_HOST: $(inputs.arvados_api_host)
      ARVADOS_API_TOKEN: $(inputs.arvados_api_token)
      ARVADOS_API_HOST_INSECURE: $(""+inputs.arvado_api_host_insecure)
  InlineJavascriptRequirement: {}
inputs:
  arvados_api_token: string
  arvado_api_host_insecure: boolean
  arvados_api_host: string
  check_collections: string[]
  preparescript:
    type: File
    default:
      class: File
      location: check_exist.py
    inputBinding:
      position: 1
outputs:
  success:
    type: boolean
    outputBinding:
      glob: success
      loadContents: true
      outputEval: $(self[0].contents=="true")
baseCommand: python2