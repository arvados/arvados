cwlVersion: v1.0
class: CommandLineTool
requirements:
  SchemaDefRequirement:
    types:
      - $import: FileOnCluster.yml
inputs:
  dataset:
    type: File[]
    inputBinding:
      position: 1
  merge_py:
    type: File
    default:
      class: File
      location: merge.py
outputs:
  out:
    type: File
    outputBinding:
      glob: merged.csv

arguments: [python, $(inputs.merge_py)]
