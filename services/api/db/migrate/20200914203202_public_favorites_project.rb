# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class PublicFavoritesProject < ActiveRecord::Migration[5.2]
  include CurrentApiClient
  def change
    act_as_system_user do
      public_project_group
      public_project_read_permission
      Link.where(link_class: "star",
                 owner_uuid: system_user_uuid,
                 tail_uuid: all_users_group_uuid).each do |ln|
        ln.owner_uuid = public_project_uuid
        ln.tail_uuid = public_project_uuid
        ln.save!
      end
    end
  end
end
