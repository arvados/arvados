# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class TimestampsNotNull < ActiveRecord::Migration
  def up
    ActiveRecord::Base.connection.tables.each do |t|
      next if t == 'schema_migrations'
      change_column t.to_sym, :created_at, :datetime, :null => false
      change_column t.to_sym, :updated_at, :datetime, :null => false
    end
  end
  def down
    # There might have been a NULL constraint before this, depending
    # on the version of Rails used to build the database.
  end
end
