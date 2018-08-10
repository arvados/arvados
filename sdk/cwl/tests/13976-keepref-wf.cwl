cwlVersion: v1.0
class: CommandLineTool
requirements:
  - class: InlineJavascriptRequirement
arguments:
  - ls
  - -l
  - $(inputs.hello)
inputs:
  hello:
    type: File
    default:
      class: File
      location: keep:4d8a70b1e63b2aad6984e40e338e2373+69/hello.txt
    secondaryFiles:
      - .idx
outputs: []