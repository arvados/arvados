# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

cwlVersion: v1.0
class: CommandLineTool
requirements:
  InitialWorkDirRequirement:
    listing:
      - entryname: input.json
        entry: $(JSON.stringify(inputs.obj))
      - entryname: config.json
        entry: |-
          ${
          return JSON.stringify({
            arvados_cluster_ids: inputs.arvados_cluster_ids,
            scrub_images: [inputs.scrub_image],
            scrub_collections: inputs.scrub_collections
          });
          }
  EnvVarRequirement:
    envDef:
      ARVADOS_API_HOST: $(inputs.arvados_api_host)
      ARVADOS_API_TOKEN: $(inputs.arvados_api_token)
      ARVADOS_API_HOST_INSECURE: $(""+inputs.arvado_api_host_insecure)
  InlineJavascriptRequirement: {}
hints:
  DockerRequirement:
    dockerPull: arvados/jobs
inputs:
  arvados_api_token: string
  arvado_api_host_insecure: boolean
  arvados_api_host: string
  arvados_cluster_ids: string[]
  wf: File
  obj: Any
  scrub_image: string
  scrub_collections: string[]
  preparescript:
    type: File
    default:
      class: File
      location: prepare.py
    inputBinding:
      position: 1
outputs:
  done:
    type: boolean
    outputBinding:
      outputEval: $(true)
baseCommand: python
