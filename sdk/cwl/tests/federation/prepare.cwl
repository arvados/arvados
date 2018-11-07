cwlVersion: v1.0
class: CommandLineTool
requirements:
  InitialWorkDirRequirement:
    listing:
      - entryname: input.json
        entry: $(JSON.stringify(inputs.obj))
      - entryname: clusters.json
        entry: $(JSON.stringify(inputs.arvados_cluster_ids))
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
  arvados_cluster_ids: string[]
  wf: File
  obj: Any
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
baseCommand: python2