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
