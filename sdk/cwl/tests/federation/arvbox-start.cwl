cwlVersion: v1.0
class: CommandLineTool
$namespaces:
  arv: "http://arvados.org/cwl#"
  cwltool: "http://commonwl.org/cwltool#"
inputs:
  container_name: string
  arvbox_data: Directory
outputs:
  cluster_id:
    type: string
    outputBinding:
      glob: status.txt
      loadContents: true
      outputEval: |
        ${
        var sp = self[0].contents.split("\n");
        for (var i = 0; i < sp.length; i++) {
          if (sp[i].startsWith("Cluster id: ")) {
            return sp[i].substr(12);
          }
        }
        }
  container_ip:
    type: string
    outputBinding:
      glob: status.txt
      loadContents: true
      outputEval: |
        ${
        var sp = self[0].contents.split("\n");
        for (var i = 0; i < sp.length; i++) {
          if (sp[i].startsWith("Container IP: ")) {
            return sp[i].substr(14);
          }
        }
        }
requirements:
  EnvVarRequirement:
    envDef:
      ARVBOX_CONTAINER: $(inputs.container_name)
      ARVBOX_DATA: $(inputs.arvbox_data.path)
  ShellCommandRequirement: {}
  InitialWorkDirRequirement:
    listing:
      - entry: $(inputs.arvbox_data)
        entryname: $(inputs.container_name)
        writable: true
  cwltool:InplaceUpdateRequirement:
    inplaceUpdate: true
  InlineJavascriptRequirement: {}
arguments:
  - shellQuote: false
    valueFrom: |
      arvbox start dev && arvbox status > status.txt
