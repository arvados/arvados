cwlVersion: v1.0
class: Workflow
$namespaces:
  arv: "http://arvados.org/cwl#"
  cwltool: "http://commonwl.org/cwltool#"
requirements:
  SubworkflowFeatureRequirement: {}
inputs:
  i:
    type: File
    # secondaryFiles:
    #   - .fai
    #   - .ann
    #   - .amb
outputs: []
steps:
  step1:
    in:
      i: i
    out: []
    run: sub.cwl
    requirements:
      arv:RunInSingleContainer: {}
