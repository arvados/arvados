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

  def current_default_owner
    # owner_uuid for newly created objects
    ((current_api_client_authorization &&
      current_api_client_authorization.default_owner_uuid) ||
     (current_user && current_user.default_owner_uuid) ||
     (current_user && current_user.uuid) ||
     nil)
  end

  # Where is the client connecting from?
  def current_api_client_ip_address
    Thread.current[:api_client_ip_address]
  end

  def system_user_uuid
    [Server::Application.config.uuid_prefix,
     User.uuid_prefix,
     '000000000000000'].join('-')
  end

  def system_group_uuid
    [Server::Application.config.uuid_prefix,
     Group.uuid_prefix,
     '000000000000000'].join('-')
  end

  def anonymous_group_uuid
    [Server::Application.config.uuid_prefix,
     Group.uuid_prefix,
     'anonymouspublic'].join('-')
  end

  def anonymous_user_uuid
    [Server::Application.config.uuid_prefix,
     User.uuid_prefix,
     'anonymouspublic'].join('-')
  end

  def system_user
    if not $system_user
      real_current_user = Thread.current[:user]
      Thread.current[:user] = User.new(is_admin: true,
                                       is_active: true,
                                       uuid: system_user_uuid)
      $system_user = User.where('uuid=?', system_user_uuid).first
      if !$system_user
        $system_user = User.new(uuid: system_user_uuid,
                                is_active: true,
                                is_admin: true,
                                email: 'root',
                                first_name: 'root',
                                last_name: '')
        $system_user.save!
        $system_user.reload
      end
      Thread.current[:user] = real_current_user
    end
    $system_user
  end

  def system_group
    if not $system_group
      act_as_system_user do
        ActiveRecord::Base.transaction do
          $system_group = Group.
            where(uuid: system_group_uuid).first_or_create do |g|
            g.update_attributes(name: "System group",
                                description: "System group")
            User.all.collect(&:uuid).each do |user_uuid|
              Link.create(link_class: 'permission',
                          name: 'can_manage',
                          tail_kind: 'arvados#group',
                          tail_uuid: system_group_uuid,
                          head_kind: 'arvados#user',
                          head_uuid: user_uuid)
            end
          end
        end
      end
    end
    $system_group
  end

  def all_users_group_uuid
    [Server::Application.config.uuid_prefix,
     Group.uuid_prefix,
     'fffffffffffffff'].join('-')
  end

  def all_users_group
    if not $all_users_group
      act_as_system_user do
        ActiveRecord::Base.transaction do
          $all_users_group = Group.
            where(uuid: all_users_group_uuid).first_or_create do |g|
            g.update_attributes(name: "All users",
                                description: "All users",
                                group_class: "role")
          end
        end
      end
    end
    $all_users_group
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
    end
  end

  def anonymous_group
    if not $anonymous_group
      act_as_system_user do
        ActiveRecord::Base.transaction do
          $anonymous_group = Group.
          where(uuid: anonymous_group_uuid).first_or_create do |g|
            g.update_attributes(name: "Anonymous group",
                                description: "Anonymous group")
          end
        end
      end
    end
    $anonymous_group
  end

  def anonymous_user
    if not $anonymous_user
      act_as_system_user do
        $anonymous_user = User.where('uuid=?', anonymous_user_uuid).first
        if !$anonymous_user
          $anonymous_user = User.new(uuid: anonymous_user_uuid,
                                     is_active: false,
                                     is_admin: false,
                                     email: 'anonymouspublic',
                                     first_name: 'anonymouspublic',
                                     last_name: 'anonymouspublic')
          $anonymous_user.save!
          $anonymous_user.reload
        end

        group_perms = Link.where(tail_uuid: anonymous_user_uuid,
                                 head_uuid: anonymous_group_uuid,
                                 link_class: 'permission',
                                 name: 'can_read')

        if !group_perms.any?
          group_perm = Link.create!(tail_uuid: anonymous_user_uuid,
                                    head_uuid: anonymous_group_uuid,
                                    link_class: 'permission',
                                    name: 'can_read')
        end
      end
    end
    $anonymous_user
  end

  def empty_collection_uuid
    'd41d8cd98f00b204e9800998ecf8427e+0'
  end

  def empty_collection
    if not $empty_collection
      act_as_system_user do
        ActiveRecord::Base.transaction do
          $empty_collection = Collection.
            where(portable_data_hash: empty_collection_uuid).
            first_or_create!(manifest_text: '', owner_uuid: anonymous_group.uuid)
        end
      end
    end
    $empty_collection
  end
end
