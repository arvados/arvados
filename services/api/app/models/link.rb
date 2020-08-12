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

  before_create :permission_to_attach_to_objects
  before_update :permission_to_attach_to_objects
  after_update :maybe_invalidate_permissions_cache
  after_create :maybe_invalidate_permissions_cache
  after_destroy :maybe_invalidate_permissions_cache
  validate :name_links_are_obsolete

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

  def check_readable_uuid attr, attr_value
    if attr == 'tail_uuid' &&
       !attr_value.nil? &&
       self.link_class == 'permission' &&
       attr_value[0..4] != Rails.configuration.ClusterID &&
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

    # Administrators can grant permissions
    return true if current_user.is_admin

    head_obj = ArvadosModel.find_by_uuid(head_uuid)

    if head_obj.nil?
      errors.add(:head_uuid, "does not exist")
      return false
    end

    # No permission links can be pointed to past collection versions
    return false if head_obj.is_a?(Collection) && head_obj.current_version_uuid != head_uuid

    # All users can grant permissions on objects they own or can manage
    return true if current_user.can?(manage: head_obj)

    # Default = deny.
    false
  end

  def maybe_invalidate_permissions_cache
    if self.link_class == 'permission'
      # Clearing the entire permissions cache can generate many
      # unnecessary queries if many active users are not affected by
      # this change. In such cases it would be better to search cached
      # permissions for head_uuid and tail_uuid, and invalidate the
      # cache for only those users. (This would require a browseable
      # cache.)
      User.invalidate_permissions_cache
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
