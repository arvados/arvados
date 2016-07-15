#!/usr/bin/env cwl-runner
cwlVersion: v1.0
class: CommandLineTool

hints:
  DockerRequirement:
    dockerPull: biodckr/bwa

baseCommand: [bwa, mem]

arguments:
  - {prefix: "-t", valueFrom: $(runtime.cores)}
  - {prefix: "-R", valueFrom: "@RG\tID:$(inputs.group_id)\tPL:$(inputs.PL)\tSM:$(inputs.sample_id)"}

inputs:
  reference:
    type: File
    inputBinding:
      position: 1
      valueFrom: $(self.dirname)/$(self.nameroot)
    secondaryFiles:
      - ^.ann
      - ^.amb
      - ^.pac
      - ^.sa
    doc: The index files produced by `bwa index`
  read_p1:
    type: File
    inputBinding:
      position: 2
    doc: The reads, in fastq format.
  read_p2:
    type: File?
    inputBinding:
      position: 3
    doc:  For mate paired reads, the second file (optional).
  group_id: string
  sample_id: string
  PL: string

stdout: $(inputs.read_p1.nameroot).sam

outputs:
  aligned_sam:
    type: stdout
