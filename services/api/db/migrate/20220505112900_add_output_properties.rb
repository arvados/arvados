# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddOutputProperties < ActiveRecord::Migration[5.2]
  def trgm_indexes
    {
      "container_requests" => "container_requests_trgm_text_search_idx",
    }
  end

  def up
    add_column :container_requests, :output_properties, :jsonb, default: {}
    add_column :containers, :output_properties, :jsonb, default: {}

    trgm_indexes.each do |model, indx|
      execute "DROP INDEX IF EXISTS #{indx}"
      execute "CREATE INDEX #{indx} ON #{model} USING gin((#{model.classify.constantize.full_text_trgm}) gin_trgm_ops)"
    end
  end

  def down
    remove_column :container_requests, :output_properties
    remove_column :containers, :output_properties

    trgm_indexes.each do |model, indx|
      execute "DROP INDEX IF EXISTS #{indx}"
      execute "CREATE INDEX #{indx} ON #{model} USING gin((#{model.classify.constantize.full_text_trgm}) gin_trgm_ops)"
    end
  end
end
