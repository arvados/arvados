#!/usr/bin/env cwl-runner
class: Workflow
cwlVersion: v1.0
id: '#main'
inputs:
  - id: arvados_api_hosts
    type:
      items: string
      type: array
  - id: superuser_tokens
    type:
      items: string
      type: array
  - default: arv-federation-migrate
    id: fed_migrate
    type: string
outputs:
  - id: report3
    outputSource: main_2/report3
    type: File
requirements:
  InlineJavascriptRequirement: {}
  MultipleInputFeatureRequirement: {}
  ScatterFeatureRequirement: {}
  StepInputExpressionRequirement: {}
  SubworkflowFeatureRequirement: {}
steps:
  - id: main_1
    in:
      arvados_api_hosts:
        source: arvados_api_hosts
      create_users:
        default:
          class: File
          location: create_users.py
      superuser_tokens:
        source: superuser_tokens
    out:
      - ar
    run:
      arguments:
        - python
        - $(inputs.create_users)
        - _script
      class: CommandLineTool
      id: main_1_embed
      inputs:
        - id: arvados_api_hosts
          type:
            items: string
            type: array
        - id: superuser_tokens
          type:
            items: string
            type: array
        - id: create_users
          type: File
      outputs:
        - id: ar
          outputBinding:
            outputEval: $(inputs.arvados_api_hosts)
          type:
            items: string
            type: array
      requirements:
        InitialWorkDirRequirement:
          listing:
            - entry: |
                {
                  "arvados_api_hosts": $(inputs.arvados_api_hosts),
                  "superuser_tokens": $(inputs.superuser_tokens)
                }
              entryname: _script
        InlineJavascriptRequirement: {}
  - id: main_2
    in:
      ar:
        source: main_1/ar
      arvados_api_hosts:
        source: arvados_api_hosts
      fed_migrate:
        source: fed_migrate
      host:
        valueFrom: '$(inputs.arvados_api_hosts[0])'
      superuser_tokens:
        source: superuser_tokens
      token:
        valueFrom: '$(inputs.superuser_tokens[0])'
    out:
      - report
      - report2
      - report3
      - r
    run:
      class: Workflow
      id: main_2_embed
      inputs:
        - id: ar
          type:
            items: string
            type: array
        - id: arvados_api_hosts
          type:
            items: string
            type: array
        - id: superuser_tokens
          type:
            items: string
            type: array
        - id: fed_migrate
          type: string
        - id: host
          type: Any
        - id: token
          type: Any
      outputs:
        - id: report
          outputSource: main_2_embed_1/report
          type: File
        - id: report2
          outputSource: main_2_embed_2/report2
          type: File
        - id: report3
          outputSource: main_2_embed_3/report3
          type: File
        - id: r
          outputSource: main_2_embed_4/r
          type: File
      requirements:
        - class: EnvVarRequirement
          envDef:
            ARVADOS_API_HOST: $(inputs.host)
            ARVADOS_API_TOKEN: $(inputs.token)
      steps:
        - id: main_2_embed_1
          in:
            fed_migrate:
              source: fed_migrate
            host:
              source: host
            token:
              source: token
          out:
            - report
          run:
            arguments:
              - $(inputs.fed_migrate)
              - '--report'
              - report.csv
            class: CommandLineTool
            id: main_2_embed_1_embed
            inputs:
              - id: fed_migrate
                type: string
              - id: host
                type: Any
              - id: token
                type: Any
            outputs:
              - id: report
                outputBinding:
                  glob: report.csv
                type: File
            requirements:
              InlineJavascriptRequirement: {}
        - id: main_2_embed_2
          in:
            host:
              source: host
            report:
              source: main_2_embed_1/report
            token:
              source: token
          out:
            - report2
          run:
            arguments:
              - sed
              - '-E'
              - 's/,(case[1-8])2?,/,1,/g'
            class: CommandLineTool
            id: main_2_embed_2_embed
            inputs:
              - id: report
                type: File
              - id: host
                type: Any
              - id: token
                type: Any
            outputs:
              - id: report2
                outputBinding:
                  glob: report.csv
                type: File
            requirements:
              InlineJavascriptRequirement: {}
            stdin: $(inputs.report)
            stdout: report.csv
        - id: main_2_embed_3
          in:
            fed_migrate:
              source: fed_migrate
            host:
              source: host
            report2:
              source: main_2_embed_2/report2
            token:
              source: token
          out:
            - report3
          run:
            arguments:
              - $(inputs.fed_migrate)
              - '--migrate'
              - $(inputs.report)
            class: CommandLineTool
            id: main_2_embed_3_embed
            inputs:
              - id: report2
                type: File
              - id: fed_migrate
                type: string
              - id: host
                type: Any
              - id: token
                type: Any
            outputs:
              - id: report3
                outputBinding:
                  outputEval: $(inputs.report2)
                type: File
            requirements:
              InlineJavascriptRequirement: {}
        - id: main_2_embed_4
          in:
            arvados_api_hosts:
              source: arvados_api_hosts
            check:
              default:
                class: File
                location: check.py
            host:
              source: host
            report3:
              source: main_2_embed_3/report3
            superuser_tokens:
              source: superuser_tokens
            token:
              source: token
          out:
            - r
          run:
            arguments:
              - python
              - $(inputs.check)
              - _script
            class: CommandLineTool
            id: main_2_embed_4_embed
            inputs:
              - id: report3
                type: File
              - id: host
                type: Any
              - id: token
                type: Any
              - id: arvados_api_hosts
                type:
                  items: string
                  type: array
              - id: superuser_tokens
                type:
                  items: string
                  type: array
              - id: check
                type: File
            outputs:
              - id: r
                outputBinding:
                  outputEval: $(inputs.report3)
                type: File
            requirements:
              InitialWorkDirRequirement:
                listing:
                  - entry: |
                      {
                        "arvados_api_hosts": $(inputs.arvados_api_hosts),
                        "superuser_tokens": $(inputs.superuser_tokens)
                      }
                    entryname: _script
              InlineJavascriptRequirement: {}

