cwlVersion: v1.0
class: CommandLineTool
baseCommand: echo
stdout: output.txt
$namespaces:
  arv: "http://arvados.org/cwl#"
hints:
  arv:RuntimeConstraints:
    outputDirType: local_output_dir
inputs:
  message:
    type: string
    inputBinding:
      position: 1
outputs:
  output:
    type: stdout
