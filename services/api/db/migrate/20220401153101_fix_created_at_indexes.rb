# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class FixCreatedAtIndexes < ActiveRecord::Migration[5.2]
  @@idxtables = [:collections, :container_requests, :groups, :links, :repositories, :users, :virtual_machines, :workflows, :logs]

  def up
    @@idxtables.each do |table|
      ActiveRecord::Base.connection.execute("DROP INDEX IF EXISTS index_#{table.to_s}_on_created_at")
      ActiveRecord::Base.connection.execute("DROP INDEX IF EXISTS index_#{table.to_s}_on_created_at_and_uuid")
      ActiveRecord::Base.connection.execute("DROP INDEX IF EXISTS index_#{table.to_s}_on_modified_at")
      ActiveRecord::Base.connection.execute("DROP INDEX IF EXISTS index_#{table.to_s}_on_modified_at_uuid")
      ActiveRecord::Base.connection.execute("DROP INDEX IF EXISTS index_#{table.to_s}_on_modified_at_and_uuid")

      ActiveRecord::Base.connection.execute("CREATE INDEX IF NOT EXISTS index_#{table.to_s}_on_created_at_and_uuid ON #{table.to_s} USING btree (created_at, uuid)")
      ActiveRecord::Base.connection.execute("CREATE INDEX IF NOT EXISTS index_#{table.to_s}_on_modified_at_and_uuid ON #{table.to_s} USING btree (modified_at, uuid)")
    end
  end

  def down
    @@idxtables.each do |table|
      ActiveRecord::Base.connection.execute("DROP INDEX IF EXISTS index_#{table.to_s}_on_created_at")
      ActiveRecord::Base.connection.execute("DROP INDEX IF EXISTS index_#{table.to_s}_on_created_at_and_uuid")
      ActiveRecord::Base.connection.execute("DROP INDEX IF EXISTS index_#{table.to_s}_on_modified_at")
      ActiveRecord::Base.connection.execute("DROP INDEX IF EXISTS index_#{table.to_s}_on_modified_at_uuid")
      ActiveRecord::Base.connection.execute("DROP INDEX IF EXISTS index_#{table.to_s}_on_modified_at_and_uuid")

      ActiveRecord::Base.connection.execute("CREATE INDEX IF NOT EXISTS index_#{table.to_s}_on_created_at_and_uuid ON #{table.to_s} USING btree (created_at, uuid)")
      ActiveRecord::Base.connection.execute("CREATE INDEX IF NOT EXISTS index_#{table.to_s}_on_modified_at_uuid ON #{table.to_s} USING btree (modified_at desc, uuid asc)")
    end
  end
end
