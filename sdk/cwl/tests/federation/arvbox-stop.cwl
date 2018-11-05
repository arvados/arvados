cwlVersion: v1.0
class: CommandLineTool
$namespaces:
  arv: "http://arvados.org/cwl#"
  cwltool: "http://commonwl.org/cwltool#"
inputs:
  container_name: string
outputs: []
requirements:
  EnvVarRequirement:
    envDef:
      ARVBOX_CONTAINER: $(inputs.container_name)
arguments: [arvbox, stop]