# Test case for arvados-cwl-runner
#
# Used to test whether scanning a workflow file for dependencies
# (e.g. submit_tool.cwl) and uploading to Keep works as intended.

class: Workflow
cwlVersion: v1.0
inputs:
  - id: x
    type: File
outputs: []
steps:
  - id: step1
    in:
      - { id: x, source: "#x" }
    out: []
    run: ../tool/submit_tool.cwl
