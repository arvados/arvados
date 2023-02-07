---
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

ssl_key_encrypted:
  enabled: __SSL_KEY_ENCRYPTED__
  aws_secret_name: __SSL_KEY_AWS_SECRET_NAME__
  ssl_password_file: /etc/nginx/ssl/ssl_key_password.txt
  ssl_password_connector_script: /usr/local/sbin/password_secret_connector.sh
