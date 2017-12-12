# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class FullTextSearchIndexes < ActiveRecord::Migration
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

  def replace_index(t)
    i = fts_indexes[t]
    t.classify.constantize.reset_column_information
    execute "DROP INDEX IF EXISTS #{i}"
    execute "CREATE INDEX #{i} ON #{t} USING gin(#{t.classify.constantize.full_text_tsvector})"
  end

  def up
    # remove existing fts indexes and create up to date ones with no
    # leading space
    fts_indexes.keys.each do |t|
      replace_index(t)
    end
  end

  def down
    fts_indexes.each do |t, i|
      remove_index t.to_sym, :name => i
    end
  end
end
