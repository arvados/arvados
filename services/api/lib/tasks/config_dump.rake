# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

namespace :config do
  desc 'Show site configuration'
  task dump: :environment do
    cfg = { "Clusters" => {}}
    cfg["Clusters"][$arvados_config["ClusterID"]] = $arvados_config.select {|k,v| k != "ClusterID"}
    puts cfg.to_yaml
  end
end
