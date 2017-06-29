#!/usr/bin/env ruby
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

# Install the supplied string (or a randomly generated token, if none
# is given) as an API token that authenticates to the system user
# account.
#
# Print the token on stdout.

require './lib/create_superuser_token'
include CreateSuperUserToken

supplied_token = ARGV[0]

token = CreateSuperUserToken.create_superuser_token supplied_token
puts token
