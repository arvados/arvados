#!/usr/bin/env ruby
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: CC-BY-SA-3.0

abort 'Error: Ruby >= 1.9.3 required.' if RUBY_VERSION < '1.9.3'

require 'arvados'

arv = Arvados.new(api_version: 'v1')
arv.node.list[:items].each do |node|
  if node[:crunch_worker_state] != 'down'
    ping_age = (Time.now - Time.parse(node[:last_ping_at])).to_i rescue -1
    puts "#{node[:uuid]} #{node[:crunch_worker_state]} #{ping_age}"
  end
end
