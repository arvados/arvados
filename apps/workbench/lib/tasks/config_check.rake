# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

namespace :config do
  desc 'Ensure site configuration has all required settings'
  task check: :environment do
    $application_config.sort.each do |k, v|
      if ENV.has_key?('QUIET') then
        # Make sure we still check for the variable to exist
        eval("Rails.configuration.#{k}")
      else
        if /(password|secret)/.match(k) then
          # Make sure we still check for the variable to exist, but don't print the value
          eval("Rails.configuration.#{k}")
          $stderr.puts "%-32s %s" % [k, '*********']
        else
          $stderr.puts "%-32s %s" % [k, eval("Rails.configuration.#{k}")]
        end
      end
    end
  end
end
