class: CommandLineTool
cwlVersion: v1.0
requirements:
  InitialWorkDirRequirement:
    listing:
      - $(inputs.inp1)
      - $(inputs.inp2)
      - $(inputs.inp3)
inputs:
  inp1: File
  inp2: [File, Directory]
  inp3: Directory
outputs: []
arguments: [echo, $(inputs.inp1), $(inputs.inp2), $(inputs.inp3)]
