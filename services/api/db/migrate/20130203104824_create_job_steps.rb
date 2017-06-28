# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class CreateJobSteps < ActiveRecord::Migration
  def change
    create_table :job_steps do |t|
      t.string :uuid
      t.string :owner
      t.string :modified_by_client
      t.string :modified_by_user
      t.datetime :modified_at
      t.string :job_uuid
      t.integer :sequence
      t.text :parameters
      t.text :output
      t.float :progress
      t.boolean :success

      t.timestamps
    end
    add_index :job_steps, :uuid, :unique => true
    add_index :job_steps, :job_uuid
    add_index :job_steps, :sequence
    add_index :job_steps, :success
  end
end
