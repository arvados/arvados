# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'update_permissions'

include CurrentApiClient

def fix_roles_projects
  batch_update_permissions do
    # This migration is not reversible.  However, the behavior it
    # enforces is backwards-compatible, and most of the time there
    # shouldn't be anything to do at all.
    act_as_system_user do
      ActiveRecord::Base.transaction do
        Group.where("(group_class != 'project' and group_class != 'filter') or group_class is null").each do |g|
          # 1) any group not group_class != project and != filter becomes a 'role' (both empty and invalid groups)
          old_owner = g.owner_uuid
          g.owner_uuid = system_user_uuid
          g.group_class = 'role'
          g.save_with_unique_name!

          if old_owner != system_user_uuid
            # 2) Ownership of a role becomes a can_manage link
            Link.new(link_class: 'permission',
                         name: 'can_manage',
                         tail_uuid: old_owner,
                         head_uuid: g.uuid).
              save!(validate: false)
          end
        end

        ActiveRecord::Base.descendants.reject(&:abstract_class?).each do |klass|
          next if [ApiClientAuthorization,
                   AuthorizedKey,
                   Log,
                   Group].include?(klass)
          next if !klass.columns.collect(&:name).include?('owner_uuid')

          # 3) If a role owns anything, give it to system user and it
          # becomes a can_manage link
          klass.joins("join groups on groups.uuid=#{klass.table_name}.owner_uuid and groups.group_class='role'").each do |owned|
            Link.new(link_class: 'permission',
                     name: 'can_manage',
                     tail_uuid: owned.owner_uuid,
                     head_uuid: owned.uuid).
              save!(validate: false)
            owned.owner_uuid = system_user_uuid
            owned.save_with_unique_name!
          end
        end

        Group.joins("join groups as g2 on g2.uuid=groups.owner_uuid and g2.group_class='role'").each do |owned|
          Link.new(link_class: 'permission',
                       name: 'can_manage',
                       tail_uuid: owned.owner_uuid,
                       head_uuid: owned.uuid).
            save!(validate: false)
          owned.owner_uuid = system_user_uuid
          owned.save_with_unique_name!
        end

        # 4) Projects can't have outgoing permission links.  Just
        # print a warning and delete them.
        q = ActiveRecord::Base.connection.exec_query %{
select links.uuid from links, groups where groups.uuid = links.tail_uuid and
       links.link_class = 'permission' and groups.group_class = 'project'
}
        q.each do |lu|
          ln = Link.find_by_uuid(lu['uuid'])
          puts "WARNING: Projects cannot have outgoing permission links, removing '#{ln.name}' link #{ln.uuid} from #{ln.tail_uuid} to #{ln.head_uuid}"
          Rails.logger.warn "Projects cannot have outgoing permission links, removing '#{ln.name}' link #{ln.uuid} from #{ln.tail_uuid} to #{ln.head_uuid}"
          ln.destroy!
        end
      end
    end
  end
end
