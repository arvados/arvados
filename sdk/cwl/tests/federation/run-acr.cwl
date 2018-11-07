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
  runner_remote_host:
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
      outputEval: $(JSON.parse(self[0].contents))
stdout: output.json
arguments:
  - valueFrom: --disable-reuse
    position: 2
  - valueFrom: input.json
    position: 4