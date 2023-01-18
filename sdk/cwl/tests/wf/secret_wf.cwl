# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

cwlVersion: v1.2
class: Workflow
$namespaces:
  cwltool: http://commonwl.org/cwltool#
hints:
  "cwltool:Secrets":
    secrets: [pw]
  DockerRequirement:
    dockerPull: debian:buster-slim
inputs:
  pw: string
outputs:
  out:
    type: File
    outputSource: step1/out
steps:
  step1:
    in:
      pw: pw
    out: [out]
    run: secret_job.cwl
