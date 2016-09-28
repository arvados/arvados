$graph:
- class: Workflow
  hints:
  - {class: 'http://arvados.org/cwl#RunInSingleContainer'}
  id: '#main'
  inputs:
  - {id: '#main/sleeptime', type: int}
  outputs:
  - {id: '#main/out', outputSource: '#main/sleep1/out', type: string}
  requirements:
  - {class: InlineJavascriptRequirement}
  - {class: ScatterFeatureRequirement}
  - {class: StepInputExpressionRequirement}
  - {class: SubworkflowFeatureRequirement}
  steps:
  - id: '#main/sleep1'
    in:
    - {id: '#main/sleep1/blurb', valueFrom: "${\n  return String(inputs.sleeptime)\
        \ + \"b\";\n}\n"}
    - {id: '#main/sleep1/sleeptime', source: '#main/sleeptime'}
    out: ['#main/sleep1/out']
    run:
      baseCommand: sleep
      class: CommandLineTool
      inputs:
      - id: '#main/sleep1/sleeptime'
        inputBinding: {position: 1}
        type: int
      outputs:
      - id: '#main/sleep1/out'
        outputBinding: {outputEval: out}
        type: string
cwlVersion: v1.0