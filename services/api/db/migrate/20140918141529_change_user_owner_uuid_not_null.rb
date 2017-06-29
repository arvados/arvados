# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class ChangeUserOwnerUuidNotNull < ActiveRecord::Migration
  include CurrentApiClient

  def up
    User.update_all({owner_uuid: system_user_uuid}, 'owner_uuid is null')
    change_column :users, :owner_uuid, :string, :null => false
  end

  def down
    change_column :users, :owner_uuid, :string, :null => true
  end
end
