---
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: CC-BY-SA-3.0

base:
  '*':
    - arvados
    - nginx_api_configuration	
    - nginx_controller_configuration
    - nginx_keepproxy_configuration
    - nginx_keepweb_configuration
    - nginx_passenger		
    - nginx_websocket_configuration
    - nginx_workbench2_configuration
    - nginx_workbench_configuration
    - postgresql
