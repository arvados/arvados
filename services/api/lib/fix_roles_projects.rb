# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

def fix_roles_projects
  # This migration is not reversible.  However, the behavior it
  # enforces is backwards-compatible, and most of the time there
  # shouldn't be anything to do at all.
  act_as_system_user do
    ActiveRecord::Base.transaction do
      q = ActiveRecord::Base.connection.exec_query %{
select uuid from groups limit 1
}

      # 1) any group not group_class != project becomes a 'role' (both empty and invalid groups)
      ActiveRecord::Base.connection.exec_query %{
UPDATE groups set group_class='role' where group_class != 'project' or group_class is null
    }

      Group.where(group_class: "role").each do |g|
        if g.owner_uuid != system_user_uuid
          # 2) Ownership of a role becomes a can_manage link
          Link.create!(link_class: 'permission',
                       name: 'can_manage',
                       tail_uuid: g.owner_uuid,
                       head_uuid: g.uuid)
          g.owner_uuid = system_user_uuid
          g.save_with_unique_name!
        end

        # 3) If a role owns anything, give it to system user and it
        # becomes a can_manage link
        ActiveRecord::Base.descendants.reject(&:abstract_class?).each do |klass|
          next if [ApiClientAuthorization,
                   AuthorizedKey,
                   Log].include?(klass)
          next if !klass.columns.collect(&:name).include?('owner_uuid')

          klass.where(owner_uuid: g.uuid).each do |owned|
            Link.create!(link_class: 'permission',
                         name: 'can_manage',
                         tail_uuid: g.uuid,
                         head_uuid: owned.uuid)
            owned.owner_uuid = system_user_uuid
            owned.save_with_unique_name!
          end
        end
      end

      # 4) Projects can't have outgoing permission links.  Just delete them.
      q = ActiveRecord::Base.connection.exec_query %{
select links.uuid from links, groups where groups.uuid = links.tail_uuid and
       links.link_class = 'permission' and groups.group_class = 'project'
}
      q.each do |lu|
        ln = Link.find_by_uuid(lu['uuid'])
        puts "Projects cannot have outgoing permission links, '#{ln.name}' link from #{ln.tail_uuid} to #{ln.head_uuid} will be removed"
        Rails.logger.warn "Destroying invalid permission link from project #{ln.tail_uuid} to #{ln.head_uuid}"
        ln.destroy!
      end
    end
  end
end
