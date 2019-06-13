# Demonstrate Arvados federation features.  This example searches a
# list of CSV files that are hosted on different Arvados clusters.
# For each file, send a task to the remote cluster which will scan
# file and extracts the rows where the column "select_column" has one
# of the values appearing in the "select_values" file.  The home
# cluster then runs a task which pulls the results from the remote
# clusters and merges the results to produce a final report.

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
  # file and the cluster where the task should run.
  SchemaDefRequirement:
    types:
      - $import: FileOnCluster.yml

inputs:
  select_column: string
  select_values: File

  datasets:
    type:
      type: array
      items: FileOnCluster.yml#FileOnCluster

  intermediate_projects: string[]

outputs:
  # Will produce an output file with the results of the distributed
  # analysis jobs merged together.
  joined:
    type: File
    outputSource: gather-results/out

steps:
  distributed-analysis:
    in:
      select_column: select_column
      select_values: select_values
      dataset: datasets
      intermediate_projects: intermediate_projects

    # Scatter over shards, this means creating a parallel job for each
    # element in the "shards" array.  Expressions are evaluated for
    # each element.
    scatter: [dataset, intermediate_projects]
    scatterMethod: dotproduct

    # Specify the cluster target for this task.  This means each
    # separate scatter task will execute on the cluster that was
    # specified in the "cluster" field.
    #
    # Arvados handles streaming data between clusters, for example,
    # the Docker image containing the code for a particular tool will
    # be fetched on demand, as long as it is available somewhere in
    # the federation.
    hints:
      arv:ClusterTarget:
        cluster_id: $(inputs.dataset.cluster)
        project_uuid: $(inputs.intermediate_projects)

    out: [out]
    run: extract.cwl

  # Collect the results of the distributed step and join them into a
  # single output file.  Arvados handles streaming inputs,
  # intermediate results, and outputs between clusters on demand.
  gather-results:
    in:
      dataset: distributed-analysis/out
    out: [out]
    run: merge.cwl
