# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddTrigramIndexForTextSearch < ActiveRecord::Migration[5.0]
  def trgm_indexes
    {
      "collections" => "collections_trgm_text_search_idx",
      "container_requests" => "container_requests_trgm_text_search_idx",
      "groups" => "groups_trgm_text_search_idx",
      "jobs" => "jobs_trgm_text_search_idx",
      "pipeline_instances" => "pipeline_instances_trgm_text_search_idx",
      "pipeline_templates" => "pipeline_templates_trgm_text_search_idx",
      "workflows" => "workflows_trgm_text_search_idx",
    }
  end

  def up
    trgm_indexes.each do |model, indx|
      execute "CREATE INDEX #{indx} ON #{model} USING gin((#{model.classify.constantize.full_text_trgm}) gin_trgm_ops)"
    end
  end

  def down
    trgm_indexes.each do |_, indx|
      execute "DROP INDEX IF EXISTS #{indx}"
    end
  end
end
