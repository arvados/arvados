# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

cwlVersion: v1.0
class: CommandLineTool
inputs:
  acr:
    type: string?
    default: arvados-cwl-runner
    inputBinding:
      position: 1
  arvados_api_host: string
  arvados_api_token: string
  arvado_api_host_insecure:
    type: boolean
    default: false
  runner_cluster:
    type: string?
    inputBinding:
      prefix: --submit-runner-cluster
      position: 2
  wf:
    type: File
    inputBinding:
      position: 3
  obj: Any
requirements:
  InitialWorkDirRequirement:
    listing:
      - entryname: input.json
        entry: $(JSON.stringify(inputs.obj))
  EnvVarRequirement:
    envDef:
      ARVADOS_API_HOST: $(inputs.arvados_api_host)
      ARVADOS_API_TOKEN: $(inputs.arvados_api_token)
      ARVADOS_API_HOST_INSECURE: $(""+inputs.arvado_api_host_insecure)
  InlineJavascriptRequirement: {}
outputs:
  out:
    type: Any
    outputBinding:
      glob: output.json
      loadContents: true
      #outputEval: $(JSON.parse(self[0].contents))
      outputEval: $(self[0].contents)
stdout: output.json
arguments:
  - valueFrom: --disable-reuse
    position: 2
  - valueFrom: --always-submit-runner
    position: 2
  - valueFrom: --api=containers
    position: 2
  - valueFrom: input.json
    position: 4