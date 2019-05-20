cwlVersion: v1.0
class: CommandLineTool

requirements:
  - class: InitialWorkDirRequirement
    listing:
      - entry: $(inputs.filesDir)
        writable: true

inputs:
  filesDir:
    type: Directory

outputs:
  results:
    type: Directory
    outputBinding:
      glob: .

arguments: [touch, $(inputs.filesDir.path)/blurg.txt]
