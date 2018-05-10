cwlVersion: v1.0
class: CommandLineTool
inputs:
  inp2:
    type: Directory
    default:
      class: Directory
      listing:
        - class: File
          location: "inp1/hello.txt"
outputs: []
arguments: [echo, $(inputs.inp2)]