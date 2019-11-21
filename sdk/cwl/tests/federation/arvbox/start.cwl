# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

cwlVersion: v1.0
class: CommandLineTool
$namespaces:
  arv: "http://arvados.org/cwl#"
  cwltool: "http://commonwl.org/cwltool#"
inputs:
  container_name: string
  arvbox_data: Directory
  arvbox_bin: File
  branch:
    type: string
    default: master
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
  container_host:
    type: string
    outputBinding:
      glob: status.txt
      loadContents: true
      outputEval: |
        ${
        var sp = self[0].contents.split("\n");
        for (var i = 0; i < sp.length; i++) {
          if (sp[i].startsWith("Container IP: ")) {
            return sp[i].substr(14)+":8000";
          }
        }
        }
  superuser_token:
    type: string
    outputBinding:
      glob: superuser_token.txt
      loadContents: true
      outputEval: $(self[0].contents.trim())
  arvbox_data_out:
    type: Directory
    outputBinding:
      outputEval: $(inputs.arvbox_data)
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
      set -ex
      mkdir -p $ARVBOX_DATA
      if ! test -d $ARVBOX_DATA/arvados ; then
        cd $ARVBOX_DATA
        git clone https://github.com/curoverse/arvados.git
      fi
      cd $ARVBOX_DATA/arvados
      gitver=`git rev-parse HEAD`
      git fetch
      git checkout -f $(inputs.branch)
      git pull
      pulled=`git rev-parse HEAD`
      git --no-pager log -n1 $pulled

      cd $(runtime.outdir)
      if test "$gitver" = "$pulled" ; then
        $(inputs.arvbox_bin.path) start dev
      else
        $(inputs.arvbox_bin.path) restart dev
      fi
      $(inputs.arvbox_bin.path) status > status.txt
      $(inputs.arvbox_bin.path) cat /var/lib/arvados/superuser_token > superuser_token.txt
