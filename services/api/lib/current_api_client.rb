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
    $system_user = check_cache $system_user do
      real_current_user = Thread.current[:user]
      begin
        Thread.current[:user] = User.new(is_admin: true,
                                         is_active: true,
                                         uuid: system_user_uuid)
        User.where(uuid: system_user_uuid).
          first_or_create!(is_active: true,
                           is_admin: true,
                           email: 'root',
                           first_name: 'root',
                           last_name: '')
      ensure
        Thread.current[:user] = real_current_user
      end
    end
  end

  def system_group
    $system_group = check_cache $system_group do
      act_as_system_user do
        ActiveRecord::Base.transaction do
          Group.where(uuid: system_group_uuid).
            first_or_create!(name: "System group",
                             description: "System group") do |g|
            g.save!
            User.all.collect(&:uuid).each do |user_uuid|
              Link.create!(link_class: 'permission',
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
  end

  def all_users_group_uuid
    [Server::Application.config.uuid_prefix,
     Group.uuid_prefix,
     'fffffffffffffff'].join('-')
  end

  def all_users_group
    $all_users_group = check_cache $all_users_group do
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
    end
  end

  def anonymous_group
    $anonymous_group = check_cache $anonymous_group do
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

  def anonymous_user
    $anonymous_user = check_cache $anonymous_user do
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

  def empty_collection_uuid
    'd41d8cd98f00b204e9800998ecf8427e+0'
  end

  def empty_collection
    $empty_collection = check_cache $empty_collection do
      act_as_system_user do
        ActiveRecord::Base.transaction do
          Collection.
            where(portable_data_hash: empty_collection_uuid).
            first_or_create!(manifest_text: '', owner_uuid: anonymous_group.uuid)
        end
      end
    end
  end

  private

  # If the given value is nil, or the cache has been cleared since it
  # was set, yield. Otherwise, return the given value.
  def check_cache value
    if not Rails.env.test? and
        ActionController::Base.cache_store.is_a? ActiveSupport::Cache::FileStore and
        not File.owned? ActionController::Base.cache_store.cache_path
      # If we don't own the cache dir, we're probably
      # crunch-dispatch. Whoever we are, using this cache is likely to
      # either fail or screw up the cache for someone else. So we'll
      # just assume the $globals are OK to live forever.
      #
      # The reason for making the globals expire with the cache in the
      # first place is to avoid leaking state between test cases: in
      # production, we don't expect the database seeds to ever go away
      # even when the cache is cleared, so there's no particular
      # reason to expire our global variables.
    else
      Rails.cache.fetch "CurrentApiClient.$globals" do
        value = nil
        true
      end
    end
    return value unless value.nil?
    yield
  end
end
