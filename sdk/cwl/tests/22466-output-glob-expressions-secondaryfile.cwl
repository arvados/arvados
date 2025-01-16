# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

cwlVersion: v1.2
class: CommandLineTool
label: Output glob test for bug 22466

$namespaces:
  arv: "http://arvados.org/cwl#"

requirements:
- class: ShellCommandRequirement
- class: InitialWorkDirRequirement
  listing:
  - $(inputs.input_bam)
- class: InlineJavascriptRequirement

inputs:
- id: input_bam
  label: Input bam
  type: File
- id: output_bam_name
  label: Output BAM file name
  type: string?
  default: deduped
- id: sample_id
  label: Sample ID
  type: string

outputs:
- id: metrics_file
  label: Metrics file
  doc: File to which the duplication metrics will be written.
  type: File?
  outputBinding:
    glob: '*.txt'
- id: deduped_bam
  label: Deduped BAM
  doc: The output file to which marked records will be written.
  type: File?
  secondaryFiles:
  - pattern: ^.bai
    required: false
  - pattern: .bai
    required: false
  outputBinding:
    glob: |-
      ${
          var ext = inputs.input_bam.nameext.slice(1)
          return ["*", inputs.output_bam_name, ext].join(".")
      }

arguments: [touch, fake.deduped.bam, fake.deduped.bai, metrics.txt]
