# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

namespace :config do
  desc 'Print active site configuration'
  task dump: :environment do
    combined = $arvados_config.deep_dup
    combined.update $remaining_config
    puts combined.to_yaml
  end
end
