# Test case for arvados-cwl-runner. Used to test propagation of
# various input types as script_parameters in pipeline templates.

class: Workflow
cwlVersion: v1.0
inputs:
  - id: "#fileInput"
    type: File
    label: It's a file; we expect to find some characters in it.
    doc: |
      If there were anything further to say, it would be said here,
      or here.
  - id: "#boolInput"
    type: boolean
    label: True or false?
  - id: "#floatInput"
    type: float
    label: Floats like a duck
    default: 0.1
  - id: "#optionalFloatInput"
    type: ["null", float]
outputs: []
steps:
  - id: step1
    in:
      - { id: x, source: "#fileInput" }
    out: []
    run: ../tool/submit_tool.cwl
