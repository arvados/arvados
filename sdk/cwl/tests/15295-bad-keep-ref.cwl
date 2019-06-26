cwlVersion: v1.0
class: CommandLineTool
requirements:
  - class: InlineJavascriptRequirement
arguments:
  - ls
  - -l
  - $(inputs.hello)
inputs:
  hello:
    type: File
outputs: []
