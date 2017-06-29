# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class CreateCommitAncestors < ActiveRecord::Migration
  def change
    create_table :commit_ancestors do |t|
      t.string :repository_name
      t.string :descendant, :null => false
      t.string :ancestor, :null => false
      t.boolean :is, :default => false, :null => false

      t.timestamps
    end
    add_index :commit_ancestors, [:descendant, :ancestor], :unique => true
  end
end
