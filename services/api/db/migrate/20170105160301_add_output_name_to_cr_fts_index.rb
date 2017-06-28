# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddOutputNameToCrFtsIndex < ActiveRecord::Migration
  def up
    t = "container_requests"
    i = "container_requests_full_text_search_idx"
    t.classify.constantize.reset_column_information
    ActiveRecord::Base.connection.indexes(t).each do |idx|
      if idx.name == i
        remove_index t.to_sym, :name => i
        break
      end
    end
    # By now, container_request should have the new column "output_name" so full_text_tsvector
    # would include it on its results
    execute "CREATE INDEX #{i} ON #{t} USING gin(#{t.classify.constantize.full_text_tsvector});"
  end

  def down
    t = "container_requests"
    i = "container_requests_full_text_search_idx"
    remove_index t.to_sym, :name => i
  end
end
