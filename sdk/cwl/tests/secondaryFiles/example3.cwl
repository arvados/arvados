class: CommandLineTool
cwlVersion: v1.0
inputs:
  step_input:
    type: File
    secondaryFiles:
      - .idx
    default:
      class: File
      location: hello.txt
outputs: []
baseCommand: echo
