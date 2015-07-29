class Arvados::V1::RepositoriesController < ApplicationController
  skip_before_filter :find_object_by_uuid, :only => :get_all_permissions
  skip_before_filter :render_404_if_no_object, :only => :get_all_permissions
  before_filter :admin_required, :only => :get_all_permissions

  def get_all_permissions
    # users is a map of {user_uuid => User object}
    users = {}
    # user_aks is a map of {user_uuid => array of public keys}
    user_aks = {}
    # admins is an array of user_uuids
    admins = []
    User.eager_load(:authorized_keys).find_each do |u|
      next unless u.is_active or u.uuid == anonymous_user_uuid
      users[u.uuid] = u
      user_aks[u.uuid] = u.authorized_keys.collect do |ak|
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
      # evidence is an array of {name: 'can_xxx', user_uuid: 'x-y-z'},
      # one entry for each piece of evidence we find in the permission
      # database that establishes that a user can access this
      # repository. Multiple entries can be added for a given user,
      # possibly with different access levels; these will be compacted
      # below.
      evidence = []
      repo.permissions.each do |perm|
        if ArvadosModel::resource_class_for_uuid(perm.tail_uuid) == Group
          # A group has permission. Each user who has access to this
          # group also has access to the repository. Access level is
          # min(group-to-repo permission, user-to-group permission).
          users.each do |user_uuid, user|
            perm_mask = user.group_permissions[perm.tail_uuid]
            if not perm_mask
              next
            elsif perm_mask[:manage] and perm.name == 'can_manage'
              evidence << {name: 'can_manage', user_uuid: user_uuid}
            elsif perm_mask[:write] and ['can_manage', 'can_write'].index perm.name
              evidence << {name: 'can_write', user_uuid: user_uuid}
            elsif perm_mask[:read]
              evidence << {name: 'can_read', user_uuid: user_uuid}
            end
          end
        elsif users[perm.tail_uuid]
          # A user has permission; the user exists; and either the
          # user is active, or it's the special case of the anonymous
          # user which is never "active" but is allowed to read
          # content from public repositories.
          evidence << {name: perm.name, user_uuid: perm.tail_uuid}
        end
      end
      # Owner of the repository, and all admins, can RW.
      ([repo.owner_uuid] | admins).each do |user_uuid|
        # Except: no permissions for inactive users, even if they own
        # repositories.
        next unless users[user_uuid]
        evidence << {name: 'can_write', user_uuid: user_uuid}
      end
      # Distill all the evidence about permissions on this repository
      # into one hash per user, of the form {'can_xxx' => true, ...}.
      # The hash is nil for a user who has no permissions at all on
      # this particular repository.
      evidence.each do |perm|
        user_uuid = perm[:user_uuid]
        user_perms = (@repo_info[repo.uuid][:user_permissions][user_uuid] ||= {})
        user_perms[perm[:name]] = true
      end
    end
    # Revisit each {'can_xxx' => true, ...} hash for some final
    # cleanup to make life easier for the requestor.
    #
    # Add a 'gitolite_permissions' key alongside the 'can_xxx' keys,
    # for the convenience of the gitolite config file generator.
    #
    # Add all lesser permissions when a greater permission is
    # present. If the requestor only wants to know who can write, it
    # only has to test for 'can_write' in the response.
    @repo_info.values.each do |repo|
      repo[:user_permissions].each do |user_uuid, user_perms|
        if user_perms['can_manage']
          user_perms['gitolite_permissions'] = 'RW'
          user_perms['can_write'] = true
          user_perms['can_read'] = true
        elsif user_perms['can_write']
          user_perms['gitolite_permissions'] = 'RW'
          user_perms['can_read'] = true
        elsif user_perms['can_read']
          user_perms['gitolite_permissions'] = 'R'
        end
      end
    end
    # The response looks like
    #   {"kind":"...",
    #    "repositories":[r1,r2,r3,...],
    #    "user_keys":usermap}
    # where each of r1,r2,r3 looks like
    #   {"uuid":"repo-uuid-1",
    #    "name":"username/reponame",
    #    "push_url":"...",
    #    "user_permissions":{"user-uuid-a":{"can_read":true,"gitolite_permissions":"R"}}}
    # and usermap looks like
    #   {"user-uuid-a":[{"public_key":"ssh-rsa g...","authorized_key_uuid":"ak-uuid-g"},...],
    #    "user-uuid-b":[{"public_key":"ssh-rsa h...","authorized_key_uuid":"ak-uuid-h"},...],...}
    send_json(kind: 'arvados#RepositoryPermissionSnapshot',
              repositories: @repo_info.values,
              user_keys: user_aks)
  end
end
