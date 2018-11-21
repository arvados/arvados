#
# Demonstrate Arvados federation features.  This performs a parallel
# scatter over some arbitrary number of files and federated clusters,
# then joins the results.
#
cwlVersion: v1.0
class: Workflow
$namespaces:
  # When using Arvados extensions to CWL, must declare the 'arv' namespace
  arv: "http://arvados.org/cwl#"

requirements:
  InlineJavascriptRequirement: {}
  ScatterFeatureRequirement: {}
  StepInputExpressionRequirement: {}

  DockerRequirement:
    # Replace this with your own Docker container
    dockerPull: arvados/jobs

  # Define a record type so we can conveniently associate the input
  # file, the cluster on which the file lives, and the project on that
  # cluster that will own the container requests and intermediate
  # outputs.
  SchemaDefRequirement:
    types:
      - name: FileOnCluster
        type: record
        fields:
          file: File
          cluster: string
          project: string

inputs:
  # Expect an array of FileOnCluster records (defined above)
  # as our input.
  shards:
    type:
      type: array
      items: FileOnCluster

outputs:
  # Will produce an output file with the results of the distributed
  # analysis jobs joined together.
  joined:
    type: File
    outputSource: gather-results/joined

steps:
  distributed-analysis:
    in:
      # Take "shards" array as input, we scatter over it below.
      shard: shards

      # Use an expression to extract the "file" field to assign to the
      # "inp" parameter of the tool.
      inp: {valueFrom: $(inputs.shard.file)}

    # Scatter over shards, this means creating a parallel job for each
    # element in the "shards" array.  Expressions are evaluated for
    # each element.
    scatter: shard

    # Specify the cluster target for this job.  This means each
    # separate scatter job will execute on the cluster that was
    # specified in the "cluster" field.
    #
    # Arvados handles streaming data between clusters, for example,
    # the Docker image containing the code for a particular tool will
    # be fetched on demand, as long as it is available somewhere in
    # the federation.
    hints:
      arv:ClusterTarget:
        cluster_id: $(inputs.shard.cluster)
        project_uuid: $(inputs.shard.project)

    out: [out]
    run: md5sum.cwl

  # Collect the results of the distributed step and join them into a
  # single output file.  Arvados handles streaming inputs,
  # intermediate results, and outputs between clusters on demand.
  gather-results:
    in:
      inp: distributed-analysis/out
    out: [joined]
    run: cat.cwl
