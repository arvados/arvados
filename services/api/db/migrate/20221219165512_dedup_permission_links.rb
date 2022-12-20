# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class DedupPermissionLinks < ActiveRecord::Migration[5.2]
  include CurrentApiClient
  def up
    act_as_system_user do
      rows = ActiveRecord::Base.connection.select_all("SELECT MIN(uuid) AS uuid, COUNT(uuid) AS n FROM links
        WHERE tail_uuid IS NOT NULL
         AND head_uuid IS NOT NULL
         AND link_class = 'permission'
         AND name in ('can_read', 'can_write', 'can_manage')
        GROUP BY (tail_uuid, head_uuid)
        HAVING COUNT(uuid) > 1
        FOR UPDATE")
      rows.each do |row|
        Rails.logger.debug "DedupPermissionLinks: consolidating #{row['n']} links into #{row['uuid']}"
        link = Link.find_by_uuid(row['uuid'])
        # This no-op update has the side effect that the update hooks
        # will merge the highest available permission into this one
        # and then delete the others.
        link.update_attributes!(properties: link.properties.dup)
      end

      rows = ActiveRecord::Base.connection.select_all("SELECT MIN(uuid) AS uuid, COUNT(uuid) AS n FROM links
        WHERE tail_uuid IS NOT NULL
         AND head_uuid IS NOT NULL
         AND link_class = 'permission'
         AND name = 'can_login'
        GROUP BY (tail_uuid, head_uuid, properties)
        HAVING COUNT(uuid) > 1
        FOR UPDATE")
      rows.each do |row|
        Rails.logger.debug "DedupPermissionLinks: consolidating #{row['n']} links into #{row['uuid']}"
        link = Link.find_by_uuid(row['uuid'])
        link.update_attributes!(properties: link.properties.dup)
      end
    end
  end
  def down
    # no-op -- restoring redundant records would still be redundant
  end
end
