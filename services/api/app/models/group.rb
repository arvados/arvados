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
    if is_trashed_changed? or owner_uuid_changed?
      if is_trashed == true
        ActiveRecord::Base.connection.exec_query %{
insert into trashed_groups (group_uuid, trash_at)
  select target_uuid as group_uuid, $2 as trash_at from project_subtree($1);
},
                                                 'Group.trash_subtree',
                                                 [[nil, self.uuid],
                                                  [nil, self.trash_at]]
      elsif is_trashed == false && TrashedGroup.find_by_group_uuid(self.owner_uuid).nil?
        ActiveRecord::Base.connection.exec_query %{
delete from trashed_groups where group_uuid in (select * from project_subtree_notrash($1));
},
                              'Group.untrash_subtree',
                              [[nil, self.uuid]]
      end
    end
    if uuid_changed? or owner_uuid_changed? or is_trashed_changed?
      # This can change users' permissions on other groups as well as
      # this one.
      invalidate_permissions_cache
    end
  end

  def invalidate_permissions_cache
    # Ensure a new group can be accessed by the appropriate users
    # immediately after being created.
    User.invalidate_permissions_cache self.async_permissions_update
  end

  def assign_name
    if self.new_record? and (self.name.nil? or self.name.empty?)
      self.name = self.uuid
    end
    true
  end
end
