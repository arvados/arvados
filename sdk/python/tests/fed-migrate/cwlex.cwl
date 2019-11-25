#!/usr/bin/env cwl-runner
arguments:
  - cwlex
  - '$(inputs.inp ? inputs.inp.path : inputs.inpdir.path+''/''+inputs.inpfile)'
class: CommandLineTool
cwlVersion: v1.0
id: '#main'
inputs:
  - id: inp
    type:
      - 'null'
      - File
  - id: inpdir
    type:
      - 'null'
      - Directory
  - id: inpfile
    type:
      - 'null'
      - string
  - id: outname
    type:
      - 'null'
      - string
outputs:
  - id: converted
    outputBinding:
      glob: $(outname(inputs))
    type: File
requirements:
  - class: DockerRequirement
    dockerPull: commonworkflowlanguage/cwlex
  - class: InlineJavascriptRequirement
    expressionLib:
      - |

        function outname(inputs) {
          return inputs.outname ? inputs.outname : (inputs.inp ? inputs.inp.nameroot+'.cwl' : inputs.inpfile.replace(/(.*).cwlex/, '$1.cwl'));
        }
stdout: $(outname(inputs))

