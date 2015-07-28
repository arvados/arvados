class Arvados::V1::RepositoriesController < ApplicationController
  skip_before_filter :find_object_by_uuid, :only => :get_all_permissions
  skip_before_filter :render_404_if_no_object, :only => :get_all_permissions
  before_filter :admin_required, :only => :get_all_permissions
  def get_all_permissions
    users = {}
    user_aks = {}
    admins = []
    User.eager_load(:authorized_keys).find_each do |u|
      next unless u.is_active or u.uuid == anonymous_user_uuid
      users[u.uuid] = u
      user_aks[u.uuid] ||= u.authorized_keys.
        collect do |ak|
        {
          public_key: ak.public_key,
          authorized_key_uuid: ak.uuid
        }
      end
      admins << u.uuid if u.is_admin
    end
    @repo_info = {}
    Repository.eager_load(:permissions).find_each do |repo|
      @repo_info[repo.uuid] = {
        uuid: repo.uuid,
        name: repo.name,
        push_url: repo.push_url,
        fetch_url: repo.fetch_url,
        user_permissions: {},
      }
      gitolite_permissions = ''
      perms = []
      repo.permissions.each do |perm|
        if ArvadosModel::resource_class_for_uuid(perm.tail_uuid) == Group
          users.each do |user_uuid, user|
            perm_mask = user.group_permissions[perm.tail_uuid]
            if not perm_mask
              next
            elsif perm_mask[:manage] and perm.name == 'can_manage'
              perms << {name: 'can_manage', user_uuid: user_uuid}
            elsif perm_mask[:write] and ['can_manage', 'can_write'].index perm.name
              perms << {name: 'can_write', user_uuid: user_uuid}
            elsif perm_mask[:read]
              perms << {name: 'can_read', user_uuid: user_uuid}
            end
          end
        elsif users[perm.tail_uuid]
          # user exists and is (active or the anonymous user)
          perms << {name: perm.name, user_uuid: perm.tail_uuid}
        end
      end
      # Owner of the repository, and all admins, can RW
      ([repo.owner_uuid] | admins).each do |user_uuid|
        # no permissions for inactive user, even when owner of repo
        next unless users[user_uuid]
        perms << {name: 'can_write', user_uuid: user_uuid}
      end
      perms.each do |perm|
        user_uuid = perm[:user_uuid]
        ri = (@repo_info[repo.uuid][:user_permissions][user_uuid] ||= {})
        ri[perm[:name]] = true
      end
    end
    @repo_info.values.each do |repo_users|
      repo_users[:user_permissions].each do |user_uuid,perms|
        if perms['can_manage']
          perms[:gitolite_permissions] = 'RW'
          perms['can_write'] = true
          perms['can_read'] = true
        elsif perms['can_write']
          perms[:gitolite_permissions] = 'RW'
          perms['can_read'] = true
        elsif perms['can_read']
          perms[:gitolite_permissions] = 'R'
        end
      end
    end
    send_json(kind: 'arvados#RepositoryPermissionSnapshot',
              repositories: @repo_info.values,
              user_keys: user_aks)
  end
end
