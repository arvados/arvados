# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

cwlVersion: v1.1
class: CommandLineTool
$namespaces:
  arv: "http://arvados.org/cwl#"
  cwltool: "http://commonwl.org/cwltool#"
requirements:
  EnvVarRequirement:
    envDef:
      ARVADOS_API_HOST: $(inputs.container_host)
      ARVADOS_API_TOKEN: $(inputs.superuser_token)
      ARVADOS_API_HOST_INSECURE: "true"
  LoadListingRequirement:
    loadListing: no_listing
  InlineJavascriptRequirement: {}
  InplaceUpdateRequirement:
    inplaceUpdate: true
  DockerRequirement:
    dockerPull: arvados/jobs
  NetworkAccess:
    networkAccess: true
inputs:
  container_host: string
  superuser_token: string
  make_user_script:
    type: File
    default:
      class: File
      location: setup_user.py
outputs:
  test_user_uuid: string
  test_user_token: string
arguments: [python, $(inputs.make_user_script)]
