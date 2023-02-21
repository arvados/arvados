---
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

ssl_key_encrypted:
  enabled: __SSL_KEY_ENCRYPTED__
  aws_secret_name: __SSL_KEY_AWS_SECRET_NAME__
  aws_region: __SSL_KEY_AWS_REGION__
  privkey_password_filename: ssl-privkey-password
  privkey_password_script: /usr/local/sbin/password_secret_connector.sh
