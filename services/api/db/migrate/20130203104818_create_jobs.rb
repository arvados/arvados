# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class CreateJobs < ActiveRecord::Migration
  def change
    create_table :jobs do |t|
      t.string :uuid
      t.string :owner
      t.string :modified_by_client
      t.string :modified_by_user
      t.datetime :modified_at
      t.string :submit_id
      t.string :command
      t.string :command_version
      t.text :command_parameters
      t.string :cancelled_by_client
      t.string :cancelled_by_user
      t.datetime :cancelled_at
      t.datetime :started_at
      t.datetime :finished_at
      t.boolean :running
      t.boolean :success
      t.string :output

      t.timestamps
    end
    add_index :jobs, :uuid, :unique => true
    add_index :jobs, :submit_id, :unique => true
    add_index :jobs, :command
    add_index :jobs, :finished_at
    add_index :jobs, :started_at
    add_index :jobs, :output
  end
end
