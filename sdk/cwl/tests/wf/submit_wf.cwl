# Test case for arvados-cwl-runner
#
# Used to test whether scanning a workflow file for dependencies
# (e.g. submit_tool.cwl) and uploading to Keep works as intended.

class: Workflow
inputs:
  - id: x
    type: File
outputs: []
steps:
  - id: step1
    inputs:
      - { id: x, source: "#x" }
    outputs: []
    run: ../tool/submit_tool.cwl
