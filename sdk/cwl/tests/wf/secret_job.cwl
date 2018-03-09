cwlVersion: v1.0
class: CommandLineTool
$namespaces:
  cwltool: http://commonwl.org/cwltool#
hints:
  "cwltool:Secrets":
    secrets: [pw]
requirements:
  InitialWorkDirRequirement:
    listing:
      - entryname: example.conf
        entry: |
          username: user
          password: $(inputs.pw)
inputs:
  pw: string
outputs:
  out: stdout
arguments: [cat, example.conf]
