class: Workflow
cwlVersion: v1.0
$namespaces:
  arv: "http://arvados.org/cwl#"
inputs:
  sleeptime:
    type: int[]
    default: [5]
outputs:
  out:
    type: string[]
    outputSource: scatterstep/out
requirements:
  SubworkflowFeatureRequirement: {}
  ScatterFeatureRequirement: {}
  InlineJavascriptRequirement: {}
  StepInputExpressionRequirement: {}
steps:
  scatterstep:
    in:
      sleeptime: sleeptime
    out: [out]
    scatter: sleeptime
    hints:
      - class: arv:RunInSingleContainer
    run:
      class: Workflow
      id: mysub
      inputs:
        sleeptime: int
      outputs:
        out:
          type: string
          outputSource: sleep1/out
      steps:
        sleep1:
          in:
            sleeptime: sleeptime
            blurb:
              valueFrom: |
                ${
                  return String(inputs.sleeptime) + "b";
                }
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
