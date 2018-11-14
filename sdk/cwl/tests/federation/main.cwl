#!/usr/bin/env cwl-runner
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

cwlVersion: v1.0
class: Workflow
$namespaces:
  arv: "http://arvados.org/cwl#"
  cwltool: "http://commonwl.org/cwltool#"
hints:
  cwltool:Secrets:
    secrets: [arvados_api_token]
requirements:
  StepInputExpressionRequirement: {}
  InlineJavascriptRequirement: {}
  SubworkflowFeatureRequirement: {}
inputs:
  arvados_api_token: string
  arvado_api_host_insecure:
    type: boolean
    default: false
  arvados_api_hosts: string[]
  arvados_cluster_ids: string[]
  acr: string?
  testcases:
    type: string[]
    default:
      - base-case
      - runner-home-step-remote
      - runner-remote-step-home
outputs:
  base-case-success:
    type: Any
    outputSource: base-case/success
  runner-home-step-remote-success:
    type: Any
    outputSource: runner-home-step-remote/success
  runner-remote-step-home-success:
    type: Any
    outputSource: runner-remote-step-home/success
  remote-case-success:
    type: Any
    outputSource: remote-case/success
  twostep-home-to-remote-success:
    type: Any
    outputSource: twostep-home-to-remote/success
  twostep-remote-to-home-success:
    type: Any
    outputSource: twostep-remote-to-home/success
  twostep-both-remote-success:
    type: Any
    outputSource: twostep-both-remote/success

steps:
  base-case:
    doc: |
      Base case (no federation), single step workflow with both the
      runner and step on the same cluster.
    in:
      arvados_api_token: arvados_api_token
      arvado_api_host_insecure: arvado_api_host_insecure
      arvados_api_hosts: arvados_api_hosts
      arvados_cluster_ids: arvados_cluster_ids
      acr: acr
      wf:
        default:
          class: File
          location: cases/base-case.cwl
          secondaryFiles:
            - class: File
              location: cases/md5sum.cwl
      obj:
        default:
          inp:
            class: File
            location: data/base-case-input.txt
        valueFrom: |-
          ${
          self["runOnCluster"] = inputs.arvados_cluster_ids[0];
          return self;
          }
      scrub_image: {default: "arvados/fed-test:base-case"}
      scrub_collections:
        default:
          - 031a4ced0aa99de90fb630568afc6e9b+67   # input collection
          - eb93a6718eb1a1a8ee9f66ee7d683472+51   # md5sum output collection
          - f654d4048612135f4a5e7707ec0fcf3e+112  # final output json
    out: [out, success]
    run: framework/testcase.cwl

  runner-home-step-remote:
    doc: |
      Single step workflow with the runner on the home cluster and the
      step on the remote cluster.
    in:
      arvados_api_token: arvados_api_token
      arvado_api_host_insecure: arvado_api_host_insecure
      arvados_api_hosts: arvados_api_hosts
      arvados_cluster_ids: arvados_cluster_ids
      acr: acr
      wf:
        default:
          class: File
          location: cases/runner-home-step-remote.cwl
          secondaryFiles:
            - class: File
              location: cases/md5sum.cwl
      obj:
        default:
          inp:
            class: File
            location: data/runner-home-step-remote-input.txt
        valueFrom: |-
          ${
          self["runOnCluster"] = inputs.arvados_cluster_ids[1];
          return self;
          }
      runner_cluster: { valueFrom: "$(inputs.arvados_cluster_ids[0])" }
      scrub_image: {default: "arvados/fed-test:runner-home-step-remote"}
      scrub_collections:
        default:
          - 3bc373e38751fe13dcbd62778d583242+81   # input collection
          - 428e6d91e41a3af3ae287b453949e7fd+51   # md5sum output collection
          - a4b0ddd866525655e8480f83a1ca83c6+112  # runner output json
    out: [out, success]
    run: framework/testcase.cwl

  runner-remote-step-home:
    doc: |
      Single step workflow with the runner on the remote cluster and the
      step on the home cluster.
    in:
      arvados_api_token: arvados_api_token
      arvado_api_host_insecure: arvado_api_host_insecure
      arvados_api_hosts: arvados_api_hosts
      arvados_cluster_ids: arvados_cluster_ids
      acr: acr
      wf:
        default:
          class: File
          location: cases/runner-remote-step-home.cwl
          secondaryFiles:
            - class: File
              location: cases/md5sum.cwl
      obj:
        default:
          inp:
            class: File
            location: data/runner-remote-step-home-input.txt
        valueFrom: |-
          ${
          self["runOnCluster"] = inputs.arvados_cluster_ids[0];
          return self;
          }
      runner_cluster: { valueFrom: "$(inputs.arvados_cluster_ids[1])" }
      scrub_image: {default: "arvados/fed-test:runner-remote-step-home"}
      scrub_collections:
        default:
          - 25fe10d8e8530329a738de69d9bc8ab5+81   # input collection
          - 7f052d1a04b851b6f73fba77c7802e1d+51   # md5sum output collection
          - ecb639201f454b6493757f5117f540df+112  # runner output json
    out: [out, success]
    run: framework/testcase.cwl

  remote-case:
    doc: |
      Single step workflow with both the runner and the step on the
      remote cluster.
    in:
      arvados_api_token: arvados_api_token
      arvado_api_host_insecure: arvado_api_host_insecure
      arvados_api_hosts: arvados_api_hosts
      arvados_cluster_ids: arvados_cluster_ids
      acr: acr
      wf:
        default:
          class: File
          location: cases/remote-case.cwl
          secondaryFiles:
            - class: File
              location: cases/md5sum.cwl
      obj:
        default:
          inp:
            class: File
            location: data/remote-case-input.txt
        valueFrom: |-
          ${
          self["runOnCluster"] = inputs.arvados_cluster_ids[1];
          return self;
          }
      runner_cluster: { valueFrom: "$(inputs.arvados_cluster_ids[1])" }
      scrub_image: {default: "arvados/fed-test:remote-case"}
      scrub_collections:
        default:
          - fccd49fdef8e452295f718208abafd88+69   # input collection
          - 58c0e8ea6b148134ef8577ee11307eec+51   # md5sum output collection
          - 1fd679c5ab64c123b9764024dbf560f0+112  # final output json
    out: [out, success]
    run: framework/testcase.cwl

  twostep-home-to-remote:
    doc: |
      Two step workflow.  The runner is on the home cluster, the first
      step is on the home cluster, the second step is on the remote
      cluster.
    in:
      arvados_api_token: arvados_api_token
      arvado_api_host_insecure: arvado_api_host_insecure
      arvados_api_hosts: arvados_api_hosts
      arvados_cluster_ids: arvados_cluster_ids
      acr: acr
      wf:
        default:
          class: File
          location: cases/twostep-home-to-remote.cwl
          secondaryFiles:
            - class: File
              location: cases/md5sum.cwl
            - class: File
              location: cases/rev.cwl
      obj:
        default:
          inp:
            class: File
            location: data/twostep-home-to-remote.txt
        valueFrom: |-
          ${
          self["md5sumCluster"] = inputs.arvados_cluster_ids[0];
          self["revCluster"] = inputs.arvados_cluster_ids[1];
          return self;
          }
      runner_cluster: { valueFrom: "$(inputs.arvados_cluster_ids[0])" }
      scrub_image: {default: "arvados/fed-test:twostep-home-to-remote"}
      scrub_collections:
        default:
          - 268a54947fb75115cfe05bb54cc62c30+74   # input collection
          - 400f03b8c5d2dc3dcb513a21b626ef88+51   # md5sum output collection
          - 3738166916ca5f6f6ad12bf7e06b4a21+51   # rev output collection
          - bc37c17a37aa25229e5de1339b27fbcc+112  # runner output json
    out: [out, success]
    run: framework/testcase.cwl

  twostep-remote-to-home:
    doc: |
      Two step workflow.  The runner is on the home cluster, the first
      step is on the remote cluster, the second step is on the home
      cluster.
    in:
      arvados_api_token: arvados_api_token
      arvado_api_host_insecure: arvado_api_host_insecure
      arvados_api_hosts: arvados_api_hosts
      arvados_cluster_ids: arvados_cluster_ids
      acr: acr
      wf:
        default:
          class: File
          location: cases/twostep-remote-to-home.cwl
          secondaryFiles:
            - class: File
              location: cases/md5sum.cwl
            - class: File
              location: cases/rev.cwl
      obj:
        default:
          inp:
            class: File
            location: data/twostep-remote-to-home.txt
        valueFrom: |-
          ${
          self["md5sumCluster"] = inputs.arvados_cluster_ids[1];
          self["revCluster"] = inputs.arvados_cluster_ids[0];
          return self;
          }
      runner_cluster: { valueFrom: "$(inputs.arvados_cluster_ids[0])" }
      scrub_image: {default: "arvados/fed-test:twostep-remote-to-home"}
      scrub_collections:
        default:
          - cce89b9f7b6e163978144051ce5f071a+74   # input collection
          - 0c358c3af63644c6343766feff1b7238+51   # md5sum output collection
          - 33fb7d512bf21f04847eca58cea46e74+51   # rev output collection
          - 912e04aa3db04aba008cf5cd46c277b2+112  # runner output json
    out: [out, success]
    run: framework/testcase.cwl

  twostep-both-remote:
    doc: |
      Two step workflow.  The runner is on the home cluster, both steps are
      on the remote cluster.
    in:
      arvados_api_token: arvados_api_token
      arvado_api_host_insecure: arvado_api_host_insecure
      arvados_api_hosts: arvados_api_hosts
      arvados_cluster_ids: arvados_cluster_ids
      acr: acr
      wf:
        default:
          class: File
          location: cases/twostep-both-remote.cwl
          secondaryFiles:
            - class: File
              location: cases/md5sum.cwl
            - class: File
              location: cases/rev.cwl
      obj:
        default:
          inp:
            class: File
            location: data/twostep-both-remote.txt
        valueFrom: |-
          ${
          self["md5sumCluster"] = inputs.arvados_cluster_ids[1];
          self["revCluster"] = inputs.arvados_cluster_ids[1];
          return self;
          }
      runner_cluster: { valueFrom: "$(inputs.arvados_cluster_ids[0])" }
      scrub_image: {default: "arvados/fed-test:twostep-both-remote"}
      scrub_collections:
        default:
          - 3c5e39939cf197d304ac1eac20841238+71   # input collection
          - 3edb99aa607731593969cdab663d65b4+51   # md5sum output collection
          - a91625b7139e60fe61a88cae42fbee13+51   # rev output collection
          - ddfa58a81953dad08436d571615dd584+112  # runner output json
    out: [out, success]
    run: framework/testcase.cwl

  twostep-remote-copy-to-home:
    doc: |
      Two step workflow.  The runner is on the home cluster, the first
      step is on the remote cluster, the second step is on the home
      cluster, and propagates its input file directly from input to
      output by symlinking the input file in the output directory.
      Tests that crunch-run will copy blocks from remote to local
      when preparing output collection.
    in:
      arvados_api_token: arvados_api_token
      arvado_api_host_insecure: arvado_api_host_insecure
      arvados_api_hosts: arvados_api_hosts
      arvados_cluster_ids: arvados_cluster_ids
      acr: acr
      wf:
        default:
          class: File
          location: cases/twostep-remote-copy-to-home.cwl
          secondaryFiles:
            - class: File
              location: cases/md5sum.cwl
            - class: File
              location: cases/rev-input-to-output.cwl
      obj:
        default:
          inp:
            class: File
            location: data/twostep-remote-copy-to-home.txt
        valueFrom: |-
          ${
          self["md5sumCluster"] = inputs.arvados_cluster_ids[1];
          self["revCluster"] = inputs.arvados_cluster_ids[0];
          return self;
          }
      runner_cluster: { valueFrom: "$(inputs.arvados_cluster_ids[0])" }
      scrub_image: {default: "arvados/fed-test:twostep-remote-copy-to-home"}
      scrub_collections:
        default:
          - 538887bc29a3098bf79abdb8536d17bd+79   # input collection
          - 14da0e0d52d7ab2945427074b275e9ee+51   # md5sum output collection
          - 2d3a4a840077390a0d7788f169eaba89+112  # rev output collection
          - 2d3a4a840077390a0d7788f169eaba89+112  # runner output json
    out: [out, success]
    run: framework/testcase.cwl

  scatter-gather:
    doc: ""
    in:
      arvados_api_token: arvados_api_token
      arvado_api_host_insecure: arvado_api_host_insecure
      arvados_api_hosts: arvados_api_hosts
      arvados_cluster_ids: arvados_cluster_ids
      acr: acr
      wf:
        default:
          class: File
          location: cases/scatter-gather.cwl
          secondaryFiles:
            - class: File
              location: cases/md5sum.cwl
            - class: File
              location: cases/cat.cwl
      obj:
        default:
          shards:
            - class: File
              location: data/scatter-gather-s1.txt
            - class: File
              location: data/scatter-gather-s2.txt
            - class: File
              location: data/scatter-gather-s3.txt
        valueFrom: |-
          ${
          self["clusters"] = inputs.arvados_cluster_ids;
          return self;
          }
      runner_cluster: { valueFrom: "$(inputs.arvados_cluster_ids[0])" }
      scrub_image: {default: "arvados/fed-test:scatter-gather"}
      scrub_collections:
        default:
          - 99cc18329bce1b4a5fe6c4cf60477668+209  # input collection
          - 2e570e844e03c7027baad148642d726f+51   # s1 md5sum output collection
          - 61c88ee7811d0b849b5c06376eb065a6+51   # s2 md5sum output collection
          - 85aaf18d638045fe609e025d3a319b2a+51   # s3 md5sum output collection
          - ec44bcba77e65128f1a8f843d881ede4+56   # cat output collection
          - 89de265942800ae36549109969940363+117  # runner output json
    out: [out, success]
    run: framework/testcase.cwl

  threestep-remote:
    doc: ""
    in:
      arvados_api_token: arvados_api_token
      arvado_api_host_insecure: arvado_api_host_insecure
      arvados_api_hosts: arvados_api_hosts
      arvados_cluster_ids: arvados_cluster_ids
      acr: acr
      wf:
        default:
          class: File
          location: cases/threestep-remote.cwl
          secondaryFiles:
            - class: File
              location: cases/md5sum.cwl
            - class: File
              location: cases/rev-input-to-output.cwl
      obj:
        default:
          inp:
            class: File
            location: data/threestep-remote.txt
        valueFrom: |-
          ${
          self["clusterA"] = inputs.arvados_cluster_ids[0];
          self["clusterB"] = inputs.arvados_cluster_ids[1];
          self["clusterC"] = inputs.arvados_cluster_ids[2];
          return self;
          }
      runner_cluster: { valueFrom: "$(inputs.arvados_cluster_ids[0])" }
      scrub_image: {default: "arvados/fed-test:threestep-remote"}
      scrub_collections:
        default:
          - 9fbf33e62876357fe134f619865cc5a5+68   # input collection
          - 210c5f2a716f6689b04316acd4928c10+51   # md5sum output collection
          - 3abea7506269d5ebf61fb17c78bbd2af+105  # revB output
          - 9e1b3acb28949759ad07e4c9740bbaa5+113  # revC output
          - 8c86dbec7de7948871b5e168ede417e1+120  # runner output json
    out: [out, success]
    run: framework/testcase.cwl
