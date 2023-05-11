# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: CC-BY-SA-3.0

# Set to a specific SSH public key path. Default: ~/.ssh/id_rsa.pub
# pubkey_path = "/path/to/pub.key"

# Set the instance type for your nodes. Default: m5a.large
# instance_type = {
#   default = "m5a.xlarge"
#   controller = "c5a.4xlarge"
# }

# AWS secret's name which holds the SSL certificate private key's password.
# Default: "arvados-ssl-privkey-password"
# ssl_password_secret_name_suffix = "some-name-suffix"

# User for software deployment. Depends on the AMI's distro.
# Default: 'admin'
# deploy_user = "ubuntu"

# Instance AMI to use for service nodes. Default: latest from Debian 11
# instance_ami = "ami-0481e8ba7f486bd99"