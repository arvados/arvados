# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

module CurrentApiClient
  def current_user
    Thread.current[:user]
  end

  def current_api_client
    Thread.current[:api_client]
  end

  def current_api_client_authorization
    Thread.current[:api_client_authorization]
  end

  def current_api_base
    Thread.current[:api_url_base]
  end

  # Where is the client connecting from?
  def current_api_client_ip_address
    Thread.current[:api_client_ip_address]
  end

  def system_user_uuid
    [Rails.configuration.ClusterID,
     User.uuid_prefix,
     '000000000000000'].join('-')
  end

  def system_group_uuid
    [Rails.configuration.ClusterID,
     Group.uuid_prefix,
     '000000000000000'].join('-')
  end

  def anonymous_group_uuid
    [Rails.configuration.ClusterID,
     Group.uuid_prefix,
     'anonymouspublic'].join('-')
  end

  def anonymous_user_uuid
    [Rails.configuration.ClusterID,
     User.uuid_prefix,
     'anonymouspublic'].join('-')
  end

  def public_project_uuid
    [Rails.configuration.ClusterID,
     Group.uuid_prefix,
     'publicfavorites'].join('-')
  end

  def system_user
    real_current_user = Thread.current[:user]
    begin
      Thread.current[:user] = User.new(is_admin: true,
                                       is_active: true,
                                       uuid: system_user_uuid)
      $system_user = check_cache($system_user) do
        User.where(uuid: system_user_uuid).
          first_or_create!(is_active: true,
                           is_admin: true,
                           email: 'root',
                           first_name: 'root',
                           last_name: '')
      end
    ensure
      Thread.current[:user] = real_current_user
    end
  end

  def system_group
    $system_group = check_cache($system_group) do
      act_as_system_user do
        ActiveRecord::Base.transaction do
          Group.where(uuid: system_group_uuid).
            first_or_create!(name: "System group",
                             description: "System group",
                             group_class: "role") do |g|
            g.save!
            User.all.collect(&:uuid).each do |user_uuid|
              Link.create!(link_class: 'permission',
                           name: 'can_manage',
                           tail_uuid: system_group_uuid,
                           head_uuid: user_uuid)
            end
          end
        end
      end
    end
  end

  def all_users_group_uuid
    [Rails.configuration.ClusterID,
     Group.uuid_prefix,
     'fffffffffffffff'].join('-')
  end

  def all_users_group
    $all_users_group = check_cache($all_users_group) do
      act_as_system_user do
        ActiveRecord::Base.transaction do
          Group.where(uuid: all_users_group_uuid).
            first_or_create!(name: "All users",
                             description: "All users",
                             group_class: "role")
        end
      end
    end
  end

  def act_as_system_user
    if block_given?
      act_as_user system_user do
        yield
      end
    else
      Thread.current[:user] = system_user
    end
  end

  def act_as_user user
    user_was = Thread.current[:user]
    Thread.current[:user] = user
    begin
      yield
    ensure
      Thread.current[:user] = user_was
      if user_was
        user_was.forget_cached_group_perms
      end
    end
  end

  def anonymous_group
    $anonymous_group = check_cache($anonymous_group) do
      act_as_system_user do
        ActiveRecord::Base.transaction do
          Group.where(uuid: anonymous_group_uuid).
            first_or_create!(group_class: "role",
                             name: "Anonymous users",
                             description: "Anonymous users")
        end
      end
    end
  end

  def anonymous_group_read_permission
    $anonymous_group_read_permission = check_cache($anonymous_group_read_permission) do
      act_as_system_user do
        Link.where(tail_uuid: all_users_group.uuid,
                   head_uuid: anonymous_group.uuid,
                   link_class: "permission",
                   name: "can_read").first_or_create!
      end
    end
  end

  def anonymous_user
    $anonymous_user = check_cache($anonymous_user) do
      act_as_system_user do
        User.where(uuid: anonymous_user_uuid).
          first_or_create!(is_active: false,
                           is_admin: false,
                           email: 'anonymous',
                           first_name: 'Anonymous',
                           last_name: '') do |u|
          u.save!
          Link.where(tail_uuid: anonymous_user_uuid,
                     head_uuid: anonymous_group.uuid,
                     link_class: 'permission',
                     name: 'can_read').
            first_or_create!
        end
      end
    end
  end

  def public_project_group
    $public_project_group = check_cache($public_project_group) do
      act_as_system_user do
        ActiveRecord::Base.transaction do
          Group.where(uuid: public_project_uuid).
            first_or_create!(group_class: "project",
                             name: "Public favorites",
                             description: "Public favorites")
        end
      end
    end
  end

  def public_project_read_permission
    $public_project_group_read_permission = check_cache($public_project_group_read_permission) do
      act_as_system_user do
        Link.where(tail_uuid: anonymous_group.uuid,
                   head_uuid: public_project_group.uuid,
                   link_class: "permission",
                   name: "can_read").first_or_create!
      end
    end
  end

  def anonymous_user_token_api_client
    $anonymous_user_token_api_client = check_cache($anonymous_user_token_api_client) do
      act_as_system_user do
        ActiveRecord::Base.transaction do
          ApiClient.find_or_create_by!(is_trusted: false, url_prefix: "", name: "AnonymousUserToken")
        end
      end
    end
  end

  def system_root_token_api_client
    $system_root_token_api_client = check_cache($system_root_token_api_client) do
      act_as_system_user do
        ActiveRecord::Base.transaction do
          ApiClient.find_or_create_by!(is_trusted: true, url_prefix: "", name: "SystemRootToken")
        end
      end
    end
  end

  def empty_collection_pdh
    'd41d8cd98f00b204e9800998ecf8427e+0'
  end

  def empty_collection
    $empty_collection = check_cache($empty_collection) do
      act_as_system_user do
        ActiveRecord::Base.transaction do
          Collection.
            where(portable_data_hash: empty_collection_pdh).
            first_or_create(manifest_text: '', owner_uuid: system_user.uuid, name: "empty collection") do |c|
            c.save!
            Link.where(tail_uuid: anonymous_group.uuid,
                       head_uuid: c.uuid,
                       link_class: 'permission',
                       name: 'can_read').
                  first_or_create!
            c
          end
        end
      end
    end
  end

  # Purge the module globals if necessary. If the cached value is
  # non-nil and the globals weren't purged, return the cached
  # value. Otherwise, call the block.
  #
  # Purge is only done in test mode.
  def check_cache(cached)
    if Rails.env != 'test'
      return (cached || yield)
    end
    t = Rails.cache.fetch "CurrentApiClient.$system_globals_reset" do
      Time.now.to_f
    end
    if t != $system_globals_reset
      reset_system_globals(t)
      yield
    else
      cached || yield
    end
  end

  def reset_system_globals(t)
    $system_globals_reset = t
    $system_user = nil
    $system_group = nil
    $all_users_group = nil
    $anonymous_group = nil
    $anonymous_group_read_permission = nil
    $anonymous_user = nil
    $public_project_group = nil
    $public_project_group_read_permission = nil
    $anonymous_user_token_api_client = nil
    $system_root_token_api_client = nil
    $empty_collection = nil
  end
  module_function :reset_system_globals
end

CurrentApiClient.reset_system_globals(0)
