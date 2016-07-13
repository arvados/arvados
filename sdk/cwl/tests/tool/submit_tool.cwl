# Test case for arvados-cwl-runner
#
# Used to test whether scanning a tool file for dependencies (e.g. default
# value blub.txt) and uploading to Keep works as intended.

class: CommandLineTool
cwlVersion: v1.0
requirements:
  - class: DockerRequirement
    dockerPull: debian:8
inputs:
  - id: x
    type: File
    default:
      class: File
      location: blub.txt
    inputBinding:
      position: 1
outputs: []
baseCommand: cat
