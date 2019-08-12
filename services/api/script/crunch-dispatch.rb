#!/usr/bin/env ruby
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

puts "**************************************
The jobs API (crunch v1) is no longer supported.  This is a stub
script that exists only to assist in a smooth upgrade.  You should
remove crunch-dispatch.rb from your init configuration.  This script
will now sleep forever.
**************************************
"

while true do
  sleep 10
end
