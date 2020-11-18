#!/usr/bin/env cwltool
cwlVersion: v1.0
class: CommandLineTool
stdout: superuser_token.txt
inputs:
  container: string
outputs:
  superuser_token:
    type: string
    outputBinding:
      glob: superuser_token.txt
      loadContents: true
      outputEval: $(self[0].contents.trim())
requirements:
  EnvVarRequirement:
    envDef:
      ARVBOX_CONTAINER: "$(inputs.container)"
  InlineJavascriptRequirement: {}
arguments: [arvbox, cat, /var/lib/arvados-arvbox/superuser_token]
