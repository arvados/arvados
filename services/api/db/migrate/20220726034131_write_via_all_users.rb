# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class WriteViaAllUsers < ActiveRecord::Migration[5.2]
  include CurrentApiClient
  def up
    changelinks(from: "can_read", to: "can_write")
  end
  def down
    changelinks(from: "can_write", to: "can_read")
  end
  def changelinks(from:, to:)
    ActiveRecord::Base.connection.exec_query(
      "update links set name=$1 where link_class=$2 and name=$3 and tail_uuid like $4 and head_uuid = $5",
      "migrate", [
        [nil, to],
        [nil, "permission"],
        [nil, from],
        [nil, "_____-tpzed-_______________"],
        [nil, all_users_group_uuid],
      ])
  end
end
