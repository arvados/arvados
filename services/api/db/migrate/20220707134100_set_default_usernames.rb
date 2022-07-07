# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class SetDefaultUsernames < ActiveRecord::Migration[5.2]
  include CurrentApiClient

  def up
    uuids = {
      'root' =>      system_user_uuid,
      'anonymous' => anonymous_user_uuid,
    }
    act_as_system_user do
      uuids.each_pair do |username, uuid|
        User.where(username: username).where.not(uuid: uuid).find_each.with_index do |user, index|
          # This should happen at most once
          user.username = "#{username}#{index+1}"
          user.save!
        end
        User.find_by(uuid: uuid).andand.update!(username: username)
      end
    end
  end

  def down
  end
end
