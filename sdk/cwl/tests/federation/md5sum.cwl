cwlVersion: v1.0
class: CommandLineTool
$namespaces:
  arv: "http://arvados.org/cwl#"
requirements:
  InlineJavascriptRequirement: {}
  DockerRequirement:
    dockerPull: debian:9
  arv:ClusterTarget:
    cluster_id: $(inputs.runOnCluster)
inputs:
  inp:
    type: File
    inputBinding: {}
  runOnCluster: string
outputs:
  hash:
    type: string
    outputBinding:
      glob: out.txt
      loadContents: true
      outputEval: $(self[0].contents.substr(0, 32))
stdout: out.txt
baseCommand: md5sum
