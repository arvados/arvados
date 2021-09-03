# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class DropFtsIndex < ActiveRecord::Migration[5.2]
  def fts_indexes
    {
      "collections" => "collections_full_text_search_idx",
      "container_requests" => "container_requests_full_text_search_idx",
      "groups" => "groups_full_text_search_idx",
      "jobs" => "jobs_full_text_search_idx",
      "pipeline_instances" => "pipeline_instances_full_text_search_idx",
      "pipeline_templates" => "pipeline_templates_full_text_search_idx",
      "workflows" => "workflows_full_text_search_idx",
    }
  end

  def up
    fts_indexes.keys.each do |t|
      i = fts_indexes[t]
      execute "DROP INDEX IF EXISTS #{i}"
    end
  end

  def down
    fts_indexes.keys.each do |t|
      i = fts_indexes[t]
      execute "CREATE INDEX #{i} ON #{t} USING gin(#{t.classify.constantize.full_text_tsvector})"
    end
  end
end
