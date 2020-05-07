# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'can_be_an_owner'
require 'trashable'

class Group < ArvadosModel
  include HasUuid
  include KindAndEtag
  include CommonApiTemplate
  include CanBeAnOwner
  include Trashable

  # Posgresql JSONB columns should NOT be declared as serialized, Rails 5
  # already know how to properly treat them.
  attribute :properties, :jsonbHash, default: {}

  validate :ensure_filesystem_compatible_name
  after_create :invalidate_permissions_cache
  after_update :maybe_invalidate_permissions_cache
  before_create :assign_name

  api_accessible :user, extend: :common do |t|
    t.add :name
    t.add :group_class
    t.add :description
    t.add :writable_by
    t.add :delete_at
    t.add :trash_at
    t.add :is_trashed
    t.add :properties
  end

  def ensure_filesystem_compatible_name
    # project groups need filesystem-compatible names, but others
    # don't.
    super if group_class == 'project'
  end

  def maybe_invalidate_permissions_cache
    if trash_at_changed? or owner_uuid_changed?
      # The group was added or removed from the trash.
      #
      # Strategy:
      #   Determine the time this goes in the trash
      #     (or null, if it does not belong in the trash)
      #   Compute subtree to determine the time each subproject goes
      #     in the trash
      #   Remove groups that don't belong from trash
      #   Add/update groups that do belong in the trash

      temptable = "group_subtree_#{rand(2**64).to_s(10)}"
      ActiveRecord::Base.connection.exec_query %{
create temporary table #{temptable} on commit drop
as select * from project_subtree_with_trash_at($1, LEAST($2, $3)::timestamp)
},
                                               'Group.get_subtree',
                                               [[nil, self.uuid],
                                                [nil, TrashedGroup.find_by_group_uuid(self.owner_uuid).andand.trash_at],
                                                [nil, self.trash_at]]

      ActiveRecord::Base.connection.exec_query %{
delete from trashed_groups where group_uuid in (select target_uuid from #{temptable} where trash_at is NULL);
}

      ActiveRecord::Base.connection.exec_query %{
insert into trashed_groups (group_uuid, trash_at)
  select target_uuid as group_uuid, trash_at from #{temptable} where trash_at is not NULL
on conflict (group_uuid) do update set trash_at=EXCLUDED.trash_at;
}
    end

    if uuid_changed? or owner_uuid_changed?
      # This can change users' permissions on other groups as well as
      # this one.
      invalidate_permissions_cache
    end
  end

  def invalidate_permissions_cache
    # Ensure a new group can be accessed by the appropriate users
    # immediately after being created.
    User.invalidate_permissions_cache self.async_permissions_update

    if new_record? or owner_uuid_changed?
      # 1. Compute set (group, permission) implied by traversing
      #    graph starting at this group
      # 2. Find links from outside the graph that point inside
      # 3. We now have the full set of permissions for a subset of groups.
      # 3. Upsert each permission in our subset (user, group, val)
      # 4. Delete permissions from table not in our computed subset.

      temptable_perm_from_start = "perm_from_start_#{rand(2**64).to_s(10)}"
      ActiveRecord::Base.connection.exec_query %{
create temporary table #{temptable_perm_from_start} on commit drop
as select $1::varchar as starting_uuid, target_uuid, val
from search_permission_graph($1::varchar, 3::smallint)
},
                                               'Group.search_permissions',
                                               [[nil, self.uuid]]

      temptable_additional_perms = "additional_perms_#{rand(2**64).to_s(10)}"
      ActiveRecord::Base.connection.exec_query %{
create temporary table #{temptable_additional_perms} on commit drop
as select links.tail_uuid as starting_uuid, ps.target_uuid, ps.val
  from links, lateral search_permission_graph(links.head_uuid::varchar, CASE
WHEN links.name = 'can_read' THEN 1::smallint
WHEN links.name = 'can_login' THEN 1::smallint
WHEN links.name = 'can_write' THEN 2::smallint
WHEN links.name = 'can_manage' THEN 3::smallint
END) as ps
where links.link_class='permission' and
  links.tail_uuid not in (select target_uuid from #{temptable_perm_from_start}) and
  links.head_uuid in (select target_uuid from #{temptable_perm_from_start})
}

      q1 = ActiveRecord::Base.connection.exec_query "select * from #{temptable_perm_from_start}"
      q1.each do |r|
        #puts r
      end
    end
  end

  def assign_name
    if self.new_record? and (self.name.nil? or self.name.empty?)
      self.name = self.uuid
    end
    true
  end
end
