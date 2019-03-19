# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

namespace :config do
  desc 'Show site configuration'
  task dump: :environment do
    puts $arvados_config.to_yaml
  end
end
