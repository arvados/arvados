cwlVersion: v1.0
class: Workflow
$namespaces:
  arv: "http://arvados.org/cwl#"
requirements:
  SubworkflowFeatureRequirement: {}
inputs:
  inp1:
    type: File
    default:
      class: File
      location: hello1.txt
  inp2:
    type: [File, Directory]
    default:
      class: File
      basename: "hello2.txt"
      contents: "Hello world"
  inp3:
    type: [File, Directory]
    default:
      class: Directory
      basename: inp3
      listing:
        - class: File
          basename: "hello3.txt"
          contents: "hello world"
outputs: []
steps:
  step1:
    requirements:
      arv:RunInSingleContainer: {}
    in:
      inp1: inp1
      inp2: inp2
      inp3: inp3
    out: []
    run: subwf.cwl
