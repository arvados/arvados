cwlVersion: v1.0
class: CommandLineTool
inputs:
  fastq1: File
outputs:
  out: stdout
baseCommand: echo
arguments:
  - $(inputs.fastq1.size)
stdout: size.txt