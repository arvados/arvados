cwlVersion: v1.0
class: Workflow
inputs: []
outputs: []
$namespaces:
  arv: "http://arvados.org/cwl#"
steps:
  step1:
    in: []
    out: []
    run:
      id: stepid
      class: CommandLineTool
      inputs:
        inp2:
          type: Directory
          default:
            class: Directory
            location: inp1
      outputs: []
      arguments: [echo, $(inputs.inp2)]