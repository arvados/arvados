cwlVersion: v1.0
class: CommandLineTool
inputs:
  inp2:
    type: Directory
    default:
      class: Directory
      location: inp1
outputs: []
arguments: [echo, $(inputs.inp2)]