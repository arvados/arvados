# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

if not Rails.env.production? and ENV['ENABLE_PROFILING']
  require 'rack-mini-profiler'
  require 'flamegraph'
  Rack::MiniProfilerRails.initialize! Rails.application
end
