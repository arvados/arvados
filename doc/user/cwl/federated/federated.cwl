cwlVersion: v1.0
class: Workflow
$namespaces:
  arv: "http://arvados.org/cwl#"
requirements:
  InlineJavascriptRequirement: {}
  DockerRequirement:
    dockerPull: arvados/fed-test:scatter-gather
  ScatterFeatureRequirement: {}
  SchemaDefRequirement:
    types:
      - name: FileOnCluster
        type: record
        fields:
          file: File
          cluster: string
inputs:
  shards:
    type:
      type: array
      items: FileOnCluster
outputs:
  joined:
    type: File
    outputSource: gather-results/joined
steps:
  distributed-analysis:
    in:
      shards: shards
      inp: {valueFrom: $(inputs.shards.file)}
    scatter: shards
    hints:
      arv:ClusterTarget:
        cluster_id: $(inputs.shards.cluster)
    out: [out]
    run: md5sum.cwl
  gather-results:
    in:
      inp: distributed-analysis/out
    out: [joined]
    run: cat.cwl
