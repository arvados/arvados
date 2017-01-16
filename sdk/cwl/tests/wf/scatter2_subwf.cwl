cwlVersion: v1.0
$graph:
- class: Workflow
  id: '#main'
  inputs:
  - type: int
    id: '#main/sleeptime'
  outputs:
  - type: string
    outputSource: '#main/sleep1/out'
    id: '#main/out'
  steps:
  - in:
    - valueFrom: |
        ${
          return String(inputs.sleeptime) + "b";
        }
      id: '#main/sleep1/blurb'
    - source: '#main/sleeptime'
      id: '#main/sleep1/sleeptime'
    out: ['#main/sleep1/out']
    run:
      class: CommandLineTool
      inputs:
      - type: int
        inputBinding: {position: 1}
        id: '#main/sleep1/sleeptime'
      outputs:
      - type: string
        outputBinding:
          outputEval: out
        id: '#main/sleep1/out'
      baseCommand: sleep
    id: '#main/sleep1'
  requirements:
  - {class: InlineJavascriptRequirement}
  - {class: ScatterFeatureRequirement}
  - {class: StepInputExpressionRequirement}
  - {class: SubworkflowFeatureRequirement}
  hints:
  - class: http://arvados.org/cwl#RunInSingleContainer