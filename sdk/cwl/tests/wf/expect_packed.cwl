cwlVersion: v1.0
$graph:
- class: CommandLineTool
  requirements:
  - class: DockerRequirement
    dockerPull: debian:8
  inputs:
  - id: '#submit_tool.cwl/x'
    type: File
    default:
      class: File
      location: keep:99999999999999999999999999999991+99/blub.txt
    inputBinding:
      position: 1
  outputs: []
  baseCommand: cat
  id: '#submit_tool.cwl'
- class: Workflow
  inputs:
  - id: '#main/x'
    type: File
    default: {class: File, location: 'keep:99999999999999999999999999999992+99/blorp.txt',
      basename: blorp.txt}
  - id: '#main/y'
    type: Directory
    default: {class: Directory, location: 'keep:99999999999999999999999999999998+99',
      basename: 99999999999999999999999999999998+99}
  - id: '#main/z'
    type: Directory
    default: {class: Directory, basename: anonymous, listing: [{basename: renamed.txt,
          class: File, location: 'keep:99999999999999999999999999999998+99/file1.txt'}]}
  outputs: []
  steps:
  - id: '#main/step1'
    in:
    - {id: '#main/step1/x', source: '#main/x'}
    out: []
    run: '#submit_tool.cwl'
  id: '#main'
