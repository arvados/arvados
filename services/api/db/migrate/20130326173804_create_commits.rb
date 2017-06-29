# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class CreateCommits < ActiveRecord::Migration
  def change
    create_table :commits do |t|
      t.string :repository_name
      t.string :sha1
      t.string :message

      t.timestamps
    end
    add_index :commits, [:repository_name, :sha1], :unique => true
  end
end
