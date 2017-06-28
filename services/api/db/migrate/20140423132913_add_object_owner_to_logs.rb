# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddObjectOwnerToLogs < ActiveRecord::Migration
  include CurrentApiClient

  def up
    add_column :logs, :object_owner_uuid, :string
    act_as_system_user do
      Log.find_in_batches(:batch_size => 500) do |batch|
        upd = {}
        ActiveRecord::Base.transaction do
          batch.each do |log|
            if log.properties["new_attributes"]
              log.object_owner_uuid = log.properties['new_attributes']['owner_uuid']
              log.save
            elsif log.properties["old_attributes"]
              log.object_owner_uuid = log.properties['old_attributes']['owner_uuid']
              log.save
            end
          end
        end
      end
    end
  end

  def down
    remove_column :logs, :object_owner_uuid
  end
end
