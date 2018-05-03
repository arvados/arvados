cwlVersion: v1.0
class: CommandLineTool
inputs:
  inp2:
    type: Directory
outputs: []
arguments: [echo, $(inputs.inp2)]