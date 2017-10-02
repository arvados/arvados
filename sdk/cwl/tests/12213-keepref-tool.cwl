cwlVersion: v1.0
class: CommandLineTool
requirements:
  InlineJavascriptRequirement: {}
inputs:
  fastqsdir: Directory
outputs:
  out: stdout
baseCommand: [zcat]
stdout: $(inputs.fastqsdir.listing[0].nameroot).txt
arguments:
  - $(inputs.fastqsdir.listing[0].path)
  - $(inputs.fastqsdir.listing[1].path)
