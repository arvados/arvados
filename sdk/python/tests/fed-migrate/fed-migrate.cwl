#!/usr/bin/env cwl-runner
$graph:
  - class: Workflow
    cwlVersion: v1.0
    id: '#run_test'
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
                  - 's/,(case[1-8])2?,/,\1,/g'
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
                stdin: $(inputs.report.path)
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
                  - $(inputs.report2)
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
  - arguments:
      - arvbox
      - cat
      - /var/lib/arvados/superuser_token
    class: CommandLineTool
    cwlVersion: v1.0
    id: '#superuser_tok'
    inputs:
      - id: container
        type: string
    outputs:
      - id: superuser_token
        outputBinding:
          glob: superuser_token.txt
          loadContents: true
          outputEval: '$(self[0].contents.trim())'
        type: string
    requirements:
      EnvVarRequirement:
        envDef:
          ARVBOX_CONTAINER: $(inputs.container)
      InlineJavascriptRequirement: {}
    stdout: superuser_token.txt
  - class: Workflow
    id: '#main'
    inputs:
      - id: arvados_api_hosts
        type:
          items: string
          type: array
      - id: arvados_cluster_ids
        type:
          items: string
          type: array
      - id: superuser_tokens
        type:
          items: string
          type: array
      - id: arvbox_containers
        type:
          items: string
          type: array
      - default: arv-federation-migrate
        id: fed_migrate
        type: string
    outputs:
      - id: supertok
        outputSource: main_2/supertok
        type:
          items: string
          type: array
      - id: report
        outputSource: run_test_3/report3
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
          arvados_cluster_ids:
            source: arvados_cluster_ids
        out:
          - logincluster
        run:
          class: ExpressionTool
          expression: '${return {''logincluster'': (inputs.arvados_cluster_ids[0])};}'
          inputs:
            - id: arvados_cluster_ids
              type:
                items: string
                type: array
          outputs:
            - id: logincluster
              type: string
      - id: main_2
        in:
          cluster_id:
            source: arvados_cluster_ids
          container:
            source: arvbox_containers
          host:
            source: arvados_api_hosts
          logincluster:
            source: main_1/logincluster
        out:
          - supertok
        run:
          class: Workflow
          id: main_2_embed
          inputs:
            - id: container
              type: string
            - id: cluster_id
              type: string
            - id: host
              type: string
            - id: logincluster
              type: string
          outputs:
            - id: supertok
              outputSource: superuser_tok_3/superuser_token
              type: string
          requirements:
            - class: EnvVarRequirement
              envDef:
                ARVBOX_CONTAINER: $(inputs.container)
          steps:
            - id: main_2_embed_1
              in:
                cluster_id:
                  source: cluster_id
                container:
                  source: container
                logincluster:
                  source: logincluster
                set_login:
                  default:
                    class: File
                    location: set_login.py
              out:
                - c
              run:
                arguments:
                  - sh
                  - _script
                class: CommandLineTool
                id: main_2_embed_1_embed
                inputs:
                  - id: container
                    type: string
                  - id: cluster_id
                    type: string
                  - id: logincluster
                    type: string
                  - id: set_login
                    type: File
                outputs:
                  - id: c
                    outputBinding:
                      outputEval: $(inputs.container)
                    type: string
                requirements:
                  InitialWorkDirRequirement:
                    listing:
                      - entry: >
                          set -x

                          docker cp
                          $(inputs.container):/var/lib/arvados/cluster_config.yml.override
                          .

                          chmod +w cluster_config.yml.override

                          python $(inputs.set_login.path)
                          cluster_config.yml.override $(inputs.cluster_id)
                          $(inputs.logincluster)

                          docker cp cluster_config.yml.override
                          $(inputs.container):/var/lib/arvados
                        entryname: _script
                  InlineJavascriptRequirement: {}
            - id: main_2_embed_2
              in:
                c:
                  source: main_2_embed_1/c
                container:
                  source: container
                host:
                  source: host
              out:
                - d
              run:
                arguments:
                  - sh
                  - _script
                class: CommandLineTool
                id: main_2_embed_2_embed
                inputs:
                  - id: container
                    type: string
                  - id: host
                    type: string
                  - id: c
                    type: string
                outputs:
                  - id: d
                    outputBinding:
                      outputEval: $(inputs.c)
                    type: string
                requirements:
                  InitialWorkDirRequirement:
                    listing:
                      - entry: >
                          set -x

                          arvbox hotreset

                          while ! curl --fail --insecure --silent
                          https://$(inputs.host)/discovery/v1/apis/arvados/v1/rest
                          >/dev/null ; do sleep 3 ; done

                          export ARVADOS_API_HOST=$(inputs.host)

                          export ARVADOS_API_TOKEN=\$(arvbox cat
                          /var/lib/arvados/superuser_token)

                          export ARVADOS_API_HOST_INSECURE=1

                          ARVADOS_VIRTUAL_MACHINE_UUID=\$(arvbox cat
                          /var/lib/arvados/vm-uuid)

                          while ! python -c "import arvados ;
                          arvados.api().virtual_machines().get(uuid='$ARVADOS_VIRTUAL_MACHINE_UUID').execute()"
                          2>/dev/null ; do sleep 3; done
                        entryname: _script
                  InlineJavascriptRequirement: {}
            - id: superuser_tok_3
              in:
                container:
                  source: container
                d:
                  source: main_2_embed_2/d
              out:
                - superuser_token
              run: '#superuser_tok'
        scatter:
          - container
          - cluster_id
          - host
        scatterMethod: dotproduct
      - id: run_test_3
        in:
          arvados_api_hosts:
            source: arvados_api_hosts
          fed_migrate:
            source: fed_migrate
          superuser_tokens:
            source: main_2/supertok
        out:
          - report3
        run: '#run_test'
cwlVersion: v1.0

