cwlVersion: v1.0
class: Workflow
$namespaces:
  cwltool: http://commonwl.org/cwltool#
hints:
  "cwltool:Secrets":
    secrets: [pw]
  DockerRequirement:
    dockerPull: debian:8
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
