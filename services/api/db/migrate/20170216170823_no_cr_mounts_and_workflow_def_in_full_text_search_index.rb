# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class NoCrMountsAndWorkflowDefInFullTextSearchIndex < ActiveRecord::Migration
  def fts_indexes
    {
      "container_requests" => "container_requests_full_text_search_idx",
      "workflows" => "workflows_full_text_search_idx",
    }
  end

  def up
    # remove existing fts index and recreate for container_requests and workflows
    fts_indexes.each do |t, i|
      t.classify.constantize.reset_column_information
      ActiveRecord::Base.connection.indexes(t).each do |idx|
        if idx.name == i
          remove_index t.to_sym, :name => i
          break
        end
      end
      execute "CREATE INDEX #{i} ON #{t} USING gin(#{t.classify.constantize.full_text_tsvector});"
    end
  end

  def down
    fts_indexes.each do |t, i|
      t.classify.constantize.reset_column_information
      ActiveRecord::Base.connection.indexes(t).each do |idx|
        if idx.name == i
          remove_index t.to_sym, :name => i
          break
        end
      end
    end
  end
end
