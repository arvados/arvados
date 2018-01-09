cwlVersion: v1.0
class: CommandLineTool
inputs:
  i:
    type: File
    inputBinding:
      position: 1
    secondaryFiles:
      - .fai
outputs: []
arguments: [ls, $(inputs.i), $(inputs.i.path).fai]
