# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class Link < ArvadosModel
  include HasUuid
  include KindAndEtag
  include CommonApiTemplate

  # Posgresql JSONB columns should NOT be declared as serialized, Rails 5
  # already know how to properly treat them.
  attribute :properties, :jsonbHash, default: {}

  validate :name_links_are_obsolete
  before_create :permission_to_attach_to_objects
  before_update :permission_to_attach_to_objects
  after_update :call_update_permissions
  after_create :call_update_permissions
  before_destroy :clear_permissions
  after_destroy :check_permissions

  api_accessible :user, extend: :common do |t|
    t.add :tail_uuid
    t.add :link_class
    t.add :name
    t.add :head_uuid
    t.add :head_kind
    t.add :tail_kind
    t.add :properties
  end

  def head_kind
    if k = ArvadosModel::resource_class_for_uuid(head_uuid)
      k.kind
    end
  end

  def tail_kind
    if k = ArvadosModel::resource_class_for_uuid(tail_uuid)
      k.kind
    end
  end

  protected

  def permission_to_attach_to_objects
    # Anonymous users cannot write links
    return false if !current_user

    # All users can write links that don't affect permissions
    return true if self.link_class != 'permission'

    # Administrators can grant permissions
    return true if current_user.is_admin

    head_obj = ArvadosModel.find_by_uuid(head_uuid)

    # No permission links can be pointed to past collection versions
    return false if head_obj.is_a?(Collection) && head_obj.current_version_uuid != head_uuid

    # All users can grant permissions on objects they own or can manage
    return true if current_user.can?(manage: head_obj)

    # Default = deny.
    false
  end

  PERM_LEVEL = {
    'can_read' => 1,
    'can_login' => 1,
    'can_write' => 2,
    'can_manage' => 3,
  }

  def call_update_permissions
    if self.link_class == 'permission'
      update_permissions tail_uuid, head_uuid, PERM_LEVEL[name], self.uuid
    end
  end

  def clear_permissions
    if self.link_class == 'permission'
      update_permissions tail_uuid, head_uuid, REVOKE_PERM, self.uuid
    end
  end

  def check_permissions
    if self.link_class == 'permission'
      check_permissions_against_full_refresh
    end
  end

  def name_links_are_obsolete
    if link_class == 'name'
      errors.add('name', 'Name links are obsolete')
      false
    else
      true
    end
  end

  # A user is permitted to create, update or modify a permission link
  # if and only if they have "manage" permission on the object
  # indicated by the permission link's head_uuid.
  #
  # All other links are treated as regular ArvadosModel objects.
  #
  def ensure_owner_uuid_is_permitted
    if link_class == 'permission'
      ob = ArvadosModel.find_by_uuid(head_uuid)
      raise PermissionDeniedError unless current_user.can?(manage: ob)
      # All permission links should be owned by the system user.
      self.owner_uuid = system_user_uuid
      return true
    else
      super
    end
  end
end
