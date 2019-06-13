cwlVersion: v1.0
class: CommandLineTool
requirements:
  SchemaDefRequirement:
    types:
      - $import: FileOnCluster.yml
inputs:
  select_column: string
  select_values: File
  dataset: 'FileOnCluster.yml#FileOnCluster'
  extract_py:
    type: File
    default:
      class: File
      location: extract.py
outputs:
  out:
    type: File
    outputBinding:
      glob: extracted.csv

arguments: [python, $(inputs.extract_py), $(inputs.select_column), $(inputs.select_values), $(inputs.dataset.file), $(inputs.dataset.cluster)]
