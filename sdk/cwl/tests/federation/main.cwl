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
  twostep-home-to-remote-success:
    type: Any
    outputSource: twostep-home-to-remote/success
  twostep-remote-to-home-success:
    type: Any
    outputSource: twostep-remote-to-home/success
  twostep-both-remote:
    type: Any
    outputSource: twostep-both-remote/success

steps:
  base-case:
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

  twostep-home-to-remote:
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

  # also: twostep-all-remote
