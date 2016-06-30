#!/usr/bin/env cwl-runner
cwlVersion: draft-3
class: CommandLineTool

hints:
  - class: DockerRequirement
    dockerPull: biodckr/bwa
requirements:
  - class: InlineJavascriptRequirement

baseCommand: [bwa, mem]

arguments:
  - {prefix: "-t", valueFrom: $(runtime.cores)}
  - {prefix: "-R", valueFrom: "@RG\tID:$(inputs.group_id)\tPL:$(inputs.PL)\tSM:$(inputs.sample_id)"}

inputs:
  - id: reference
    type: File
    inputBinding:
      position: 1
      valueFrom: $(self.path.match(/(.*)\.[^.]+$/)[1])
    secondaryFiles:
      - ^.ann
      - ^.amb
      - ^.pac
      - ^.sa
    description: The index files produced by `bwa index`
  - id: read_p1
    type: File
    inputBinding:
      position: 2
    description: The reads, in fastq format.
  - id: read_p2
    type: ["null", File]
    inputBinding:
      position: 3
    description:  For mate paired reads, the second file (optional).
  - id: group_id
    type: string
  - id: sample_id
    type: string
  - id: PL
    type: string

stdout: $(inputs.read_p1.path.match(/\/([^/]+)\.[^/.]+$/)[1] + ".sam")

outputs:
  - id: aligned_sam
    type: File
    outputBinding:
      glob: $(inputs.read_p1.path.match(/\/([^/]+)\.[^/.]+$/)[1] + ".sam")
