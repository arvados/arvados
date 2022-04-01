# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class FixCreatedAtIndexes < ActiveRecord::Migration[5.2]
  def tables
    %w{collections links logs groups users}
  end

  def up
    tables.each do |t|
      remove_index t.to_sym, :created_at
      add_index t.to_sym, [:created_at, :uuid]
    end
  end

  def down
    tables.each do |t|
      remove_index t.to_sym, [:created_at, :uuid]
      add_index t.to_sym, :created_at
    end
  end
end
