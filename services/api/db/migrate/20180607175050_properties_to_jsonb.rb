# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require './db/migrate/20161213172944_full_text_search_indexes'

class PropertiesToJsonb < ActiveRecord::Migration

  @@tables_columns = [["nodes", "properties"],
                      ["nodes", "info"],
                      ["container_requests", "properties"],
                      ["links", "properties"]]

  def up
    @@tables_columns.each do |table, column|
      # Drop the FT index before changing column type to avoid
      # "PG::DatatypeMismatch: ERROR: COALESCE types jsonb and text
      # cannot be matched".
      ActiveRecord::Base.connection.execute "DROP INDEX IF EXISTS #{table}_full_text_search_idx"
      ActiveRecord::Base.connection.execute "ALTER TABLE #{table} ALTER COLUMN #{column} TYPE jsonb USING #{column}::jsonb"
      ActiveRecord::Base.connection.execute "CREATE INDEX #{table}_index_on_#{column} ON #{table} USING gin (#{column})"
    end
    FullTextSearchIndexes.new.replace_index("container_requests")
  end

  def down
    @@tables_columns.each do |table, column|
      ActiveRecord::Base.connection.execute "DROP INDEX IF EXISTS #{table}_index_on_#{column}"
      ActiveRecord::Base.connection.execute "ALTER TABLE #{table} ALTER COLUMN #{column} TYPE text"
    end
  end
end
