cwlVersion: v1.0
class: Workflow
requirements:
  ScatterFeatureRequirement: {}
inputs:
  dir: Directory
outputs: []
steps:
  ex:
    in:
      dir: dir
    out: [out]
    run: 12213-keepref-expr.cwl
  tool:
    in:
      fastqsdir: ex/out
    out: []
    scatter: fastqsdir
    run: 12213-keepref-tool.cwl