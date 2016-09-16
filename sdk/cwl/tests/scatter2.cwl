class: Workflow
cwlVersion: v1.0
$namespaces:
  arv: "http://arvados.org/cwl#"
inputs:
  sleeptime:
    type: int[]
    default: [44, 29, 14]
outputs: []
requirements:
  SubworkflowFeatureRequirement: {}
  ScatterFeatureRequirement: {}
  InlineJavascriptRequirement: {}
  StepInputExpressionRequirement: {}
steps:
  scatterstep:
    in:
      sleeptime: sleeptime
    out: []
    scatter: sleeptime
    hints:
      - class: arv:RunInSingleContainer
    run:
      class: Workflow
      id: mysub
      inputs:
        sleeptime: int
      outputs: []
      steps:
        sleep1:
          in:
            sleeptime: sleeptime
          out: [out]
          run:
            class: CommandLineTool
            inputs:
              sleeptime:
                type: int
                inputBinding: {position: 1}
            outputs:
              out:
                type: string
                outputBinding:
                  outputEval: "out"
            baseCommand: sleep
        sleep2:
          in:
            sleeptime:
              source: sleeptime
              valueFrom: $(self+1)
            dep: sleep1/out
          out: []
          run:
            class: CommandLineTool
            inputs:
              sleeptime:
                type: int
                inputBinding: {position: 1}
            outputs: []
            baseCommand: sleep
