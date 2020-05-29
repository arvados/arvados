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
  validate :check_group_class
  before_create :assign_name
  after_create :after_ownership_change
  after_create :update_trash

  before_update :before_ownership_change
  after_update :after_ownership_change

  after_update :update_trash
  before_destroy :clear_permissions_and_trash

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

  def check_group_class
    if group_class != 'project' && group_class != 'role'
      errors.add :group_class, "value must be one of 'project' or 'role', was '#{group_class}'"
    end
    if group_class_changed? && !group_class_was.nil?
      errors.add :group_class, "cannot be modified after record is created"
    end
  end

  def update_trash
    if trash_at_changed? or owner_uuid_changed?
      # The group was added or removed from the trash.
      #
      # Strategy:
      #   Compute project subtree, propagating trash_at to subprojects
      #   Remove groups that don't belong from trash
      #   Add/update groups that do belong in the trash

      temptable = "group_subtree_#{rand(2**64).to_s(10)}"
      ActiveRecord::Base.connection.exec_query %{
create temporary table #{temptable} on commit drop
as select * from project_subtree_with_trash_at($1, LEAST($2, $3)::timestamp)
},
                                               'Group.update_trash.select',
                                               [[nil, self.uuid],
                                                [nil, TrashedGroup.find_by_group_uuid(self.owner_uuid).andand.trash_at],
                                                [nil, self.trash_at]]

      ActiveRecord::Base.connection.exec_delete %{
delete from trashed_groups where group_uuid in (select target_uuid from #{temptable} where trash_at is NULL);
},
                                            "Group.update_trash.delete"

      ActiveRecord::Base.connection.exec_query %{
insert into trashed_groups (group_uuid, trash_at)
  select target_uuid as group_uuid, trash_at from #{temptable} where trash_at is not NULL
on conflict (group_uuid) do update set trash_at=EXCLUDED.trash_at;
},
                                            "Group.update_trash.insert"
    end
  end

  def before_ownership_change
    if owner_uuid_changed? and !self.owner_uuid_was.nil?
      MaterializedPermission.where(user_uuid: owner_uuid_was, target_uuid: uuid).delete_all
      update_permissions self.owner_uuid_was, self.uuid, REVOKE_PERM
    end
  end

  def after_ownership_change
    if owner_uuid_changed?
      update_permissions self.owner_uuid, self.uuid, CAN_MANAGE_PERM
    end
  end

  def clear_permissions_and_trash
    MaterializedPermission.where(target_uuid: uuid).delete_all
    ActiveRecord::Base.connection.exec_delete %{
delete from trashed_groups where group_uuid=$1
}, "Group.clear_permissions_and_trash", [[nil, self.uuid]]

  end

  def assign_name
    if self.new_record? and (self.name.nil? or self.name.empty?)
      self.name = self.uuid
    end
    true
  end
end
