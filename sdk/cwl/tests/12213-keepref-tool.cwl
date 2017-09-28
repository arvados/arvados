cwlVersion: v1.0
class: CommandLineTool
requirements:
  InlineJavascriptRequirement: {}
inputs:
  fastqsdir: Directory
outputs: []
baseCommand: [zcat]
arguments:
  - $(inputs.fastqsdir.listing[0].path)
  - $(inputs.fastqsdir.listing[1].path)
