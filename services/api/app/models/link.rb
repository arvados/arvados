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
  validate :permission_to_attach_to_objects
  before_update :restrict_alter_permissions
  before_update :apply_max_overlapping_permissions
  before_create :apply_max_overlapping_permissions
  after_update :delete_overlapping_permissions
  after_update :call_update_permissions
  after_create :call_update_permissions
  before_destroy :clear_permissions
  after_destroy :delete_overlapping_permissions
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

  PermLevel = {
    'can_read' => 0,
    'can_write' => 1,
    'can_manage' => 2,
  }

  def apply_max_overlapping_permissions
    return if self.link_class != 'permission' || !PermLevel[self.name]
    Link.
      lock. # select ... for update
      where(link_class: 'permission',
            tail_uuid: self.tail_uuid,
            head_uuid: self.head_uuid,
            name: PermLevel.keys).
      where('uuid <> ?', self.uuid).each do |other|
      if PermLevel[other.name] > PermLevel[self.name]
        self.name = other.name
      end
    end
  end

  def delete_overlapping_permissions
    return if self.link_class != 'permission'
    redundant = nil
    if PermLevel[self.name]
      redundant = Link.
                    lock. # select ... for update
                    where(link_class: 'permission',
                          tail_uuid: self.tail_uuid,
                          head_uuid: self.head_uuid,
                          name: PermLevel.keys).
                    where('uuid <> ?', self.uuid)
    elsif self.name == 'can_login' &&
          self.properties.respond_to?(:has_key?) &&
          self.properties.has_key?('username')
      redundant = Link.
                    lock. # select ... for update
                    where(link_class: 'permission',
                          tail_uuid: self.tail_uuid,
                          head_uuid: self.head_uuid,
                          name: 'can_login').
                    where('properties @> ?', SafeJSON.dump({'username' => self.properties['username']})).
                    where('uuid <> ?', self.uuid)
    end
    if redundant
      redundant.each do |link|
        link.clear_permissions
      end
      redundant.delete_all
    end
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

  def check_readable_uuid attr, attr_value
    if attr == 'tail_uuid' &&
       !attr_value.nil? &&
       self.link_class == 'permission' &&
       attr_value[0..4] != Rails.configuration.ClusterID &&
       ApiClientAuthorization.remote_host(uuid_prefix: attr_value[0..4]) &&
       ArvadosModel::resource_class_for_uuid(attr_value) == User
      # Permission link tail is a remote user (the user permissions
      # are being granted to), so bypass the standard check that a
      # referenced object uuid is readable by current user.
      #
      # We could do a call to the remote cluster to check if the user
      # in tail_uuid exists.  This would detect copy-and-paste errors,
      # but add another way for the request to fail, and I don't think
      # it would improve security.  It doesn't seem to be worth the
      # complexity tradeoff.
      true
    else
      super
    end
  end

  def permission_to_attach_to_objects
    # Anonymous users cannot write links
    return false if !current_user

    # All users can write links that don't affect permissions
    return true if self.link_class != 'permission'

    if PERM_LEVEL[self.name].nil?
      errors.add(:name, "is invalid permission, must be one of 'can_read', 'can_write', 'can_manage', 'can_login'")
      return false
    end

    rsc_class = ArvadosModel::resource_class_for_uuid tail_uuid
    if rsc_class == Group
      tail_obj = Group.find_by_uuid(tail_uuid)
      if tail_obj.nil?
        errors.add(:tail_uuid, "does not exist")
        return false
      end
      if tail_obj.group_class != "role"
        errors.add(:tail_uuid, "must be a user or role, was group with group_class #{tail_obj.group_class}")
        return false
      end
    elsif rsc_class != User
      errors.add(:tail_uuid, "must be a user or role")
      return false
    end

    # Administrators can grant permissions
    return true if current_user.is_admin

    head_obj = ArvadosModel.find_by_uuid(head_uuid)

    if head_obj.nil?
      errors.add(:head_uuid, "does not exist")
      return false
    end

    # No permission links can be pointed to past collection versions
    if head_obj.is_a?(Collection) && head_obj.current_version_uuid != head_uuid
      errors.add(:head_uuid, "cannot point to a past version of a collection")
      return false
    end

    # All users can grant permissions on objects they own or can manage
    return true if current_user.can?(manage: head_obj)

    # Default = deny.
    false
  end

  def restrict_alter_permissions
    return true if self.link_class != 'permission' && self.link_class_was != 'permission'

    return true if current_user.andand.uuid == system_user.uuid

    if link_class_changed? || tail_uuid_changed? || head_uuid_changed?
      raise "Can only alter permission link level"
    end
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
      current_user.forget_cached_group_perms
    end
  end

  def clear_permissions
    if self.link_class == 'permission'
      update_permissions tail_uuid, head_uuid, REVOKE_PERM, self.uuid
      current_user.forget_cached_group_perms
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
