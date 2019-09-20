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
  - id: out
    outputSource: main_2/out
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
      - out
    run:
      arguments:
        - $(inputs.fed_migrate)
        - '--report'
        - out
      class: CommandLineTool
      id: main_2_embed
      inputs:
        - id: arvados_api_hosts
          type:
            items: string
            type: array
        - id: superuser_tokens
          type:
            items: string
            type: array
        - id: ar
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
        - id: out
          outputBinding:
            glob: out
          type: File
      requirements:
        - class: EnvVarRequirement
          envDef:
            ARVADOS_API_HOST: $(inputs.host)
            ARVADOS_API_TOKEN: $(inputs.token)

