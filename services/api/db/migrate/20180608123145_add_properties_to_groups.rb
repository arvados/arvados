# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require './db/migrate/20161213172944_full_text_search_indexes'

class AddPropertiesToGroups < ActiveRecord::Migration
  def up
    add_column :groups, :properties, :jsonb, default: {}
    ActiveRecord::Base.connection.execute("CREATE INDEX group_index_on_properties ON groups USING gin (properties);")
    FullTextSearchIndexes.new.replace_index('groups')
  end

  def down
    ActiveRecord::Base.connection.execute("DROP INDEX IF EXISTS group_index_on_properties")
    remove_column :groups, :properties
  end
end
