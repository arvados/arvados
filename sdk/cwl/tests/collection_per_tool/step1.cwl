cwlVersion: v1.0
class: CommandLineTool
inputs:
  a:
    type: File
    default:
      class: File
      location: a.txt
  b:
    type: File
    default:
      class: File
      location: b.txt
outputs: []
arguments: [echo, $(inputs.a), $(inputs.b)]