cwlVersion: v1.0
class: CommandLineTool
inputs:
  container_name: string
  this_cluster: string
  cluster_ids: string[]
  cluster_hosts: string[]
  arvbox_base: Directory
outputs: []
requirements:
  EnvVarRequirement:
    envDef:
      ARVBOX_CONTAINER: $(inputs.container_name)
      ARVBOX_BASE: $(inputs.arvbox_base.path)
  InitialWorkDirRequirement:
    listing:
      cluster_config.yml.override: |
        ${
        var remoteClusters = {};
        for (var i = 0; i < cluster_ids.length; i++) {
          remoteClusters[inputs.cluster_ids[i]] = inputs.cluster_hosts[i];
        }
        return JSON.stringify({"Cluster": {inputs.this_cluster: {"RemoteClusters": remoteClusters}}});
        }
      application.yml.override: |
        ${
        var remoteClusters = {};
        for (var i = 0; i < cluster_ids.length; i++) {
          remoteClusters[inputs.cluster_ids[i]] = inputs.cluster_hosts[i];
        }
        return JSON.stringify({"development": {"remote_hosts": remoteClusters}});
        }
  ShellCommandRequirement: {}
arguments:
  - shellQuote: false
    valueFrom: |
      docker cp cluster_config.yml.override $(inputs.container_name):/var/lib/arvados
      docker cp application.yml.override $(inputs.container_name):/usr/src/arvados/services/api/config
      arvbox sv restart api
      arvbox sv restart controller
      arvbox sv restart keepstore0
      arvbox sv restart keepstore1