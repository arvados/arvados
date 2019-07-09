# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

def diff_hash base, final
  diffed = {}
  base.each do |k,v|
    bk = base[k]
    fk = final[k]
    if bk.is_a? Hash
      d = diff_hash bk, fk
      if d.length > 0
        diffed[k] = d
      end
    else
      if bk.to_yaml != fk.to_yaml
        diffed[k] = fk
      end
    end
  end
  diffed
end

namespace :config do
  desc 'Print items that differ between legacy application.yml and system config.yml'
  task diff: :environment do
    diffed = diff_hash $arvados_config_global, $arvados_config
    cfg = { "Clusters" => {}}
    cfg["Clusters"][$arvados_config["ClusterID"]] = diffed.select {|k,v| k != "ClusterID"}
    if cfg["Clusters"][$arvados_config["ClusterID"]].empty?
      puts "No migrations required for /etc/arvados/config.yml"
    else
      puts cfg.to_yaml
    end
  end

  desc 'Print config.yml after merging with legacy application.yml'
  task migrate: :environment do
    diffed = diff_hash $arvados_config_defaults, $arvados_config
    cfg = { "Clusters" => {}}
    cfg["Clusters"][$arvados_config["ClusterID"]] = diffed.select {|k,v| k != "ClusterID"}
    puts cfg.to_yaml
  end

  desc 'Print configuration as accessed through Rails.configuration'
  task dump: :environment do
    combined = $arvados_config.deep_dup
    combined.update $remaining_config
    puts combined.to_yaml
  end

  desc 'Legacy config check task -- it is a noop now'
  task check: :environment do
    # This exists so that build/rails-package-scripts/postinst.sh doesn't fail.
  end
end
