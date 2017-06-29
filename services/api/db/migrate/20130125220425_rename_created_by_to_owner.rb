# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class RenameCreatedByToOwner < ActiveRecord::Migration
  def tables
    %w{api_clients collections logs metadata nodes pipelines pipeline_invocations projects specimens users}
  end

  def up
    tables.each do |t|
      remove_column t.to_sym, :created_by_client
      rename_column t.to_sym, :created_by_user, :owner
    end
  end

  def down
    tables.reverse.each do |t|
      rename_column t.to_sym, :owner, :created_by_user
      add_column t.to_sym, :created_by_client, :string
    end
  end
end
