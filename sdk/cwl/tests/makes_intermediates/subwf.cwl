cwlVersion: v1.0
class: Workflow
inputs:
  inp1: File
  inp2: File
  inp3: Directory
outputs: []
steps:
  step1:
    in:
      inp1: inp1
      inp2: inp2
      inp3: inp3
    out: []
    run: echo.cwl
