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

  serialize :properties, Hash

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

  def maybe_invalidate_permissions_cache
    if uuid_changed? or owner_uuid_changed? or is_trashed_changed?
      # This can change users' permissions on other groups as well as
      # this one.
      invalidate_permissions_cache
    end
  end

  def invalidate_permissions_cache
    # Ensure a new group can be accessed by the appropriate users
    # immediately after being created.
    User.invalidate_permissions_cache db_current_time.to_i
  end

  def assign_name
    if self.new_record? and (self.name.nil? or self.name.empty?)
      self.name = self.uuid
    end
    true
  end

end
