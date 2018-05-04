class: Workflow
cwlVersion: v1.0
inputs:
  toplevel_input: File
outputs: []
steps:
  step1:
    in:
      step_input: toplevel_input
    out: []
    run:
      id: sub
      class: CommandLineTool
      inputs:
        step_input:
          type: File
          secondaryFiles:
            - .idx
      outputs: []
      baseCommand: echo
