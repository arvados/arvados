class ManageAccountController < ApplicationController

  def model_class
    AuthorizedKey
  end

  def index_pane_list
    %w(Manage_account)
  end

  def index    
    # repositories current user can read / write
    @repo_links = []
    Link.where(tail_uuid: current_user.uuid,
               link_class: 'permission',
               name: ['can_write', 'can_read']).
          each do |perm_link|
            @repo_links << perm_link[:head_uuid]
          end
    @repositories = Repository.where(uuid: @repo_links)

    # virtual machines the current user can login into
    @vm_logins = {}
    Link.where(tail_uuid: current_user.uuid,
               link_class: 'permission',
               name: 'can_login').
          each do |perm_link|
            if perm_link.properties.andand[:username]
              @vm_logins[perm_link.head_uuid] ||= []
              @vm_logins[perm_link.head_uuid] << perm_link.properties[:username]
            end
          end
    @virtual_machines = VirtualMachine.where(uuid: @vm_logins.keys)

    # current user's ssh keys
    filters=[["owner_uuid", "=", current_user.uuid]]
    @ssh_keys = AuthorizedKey.where(key_type: 'SSH', filters: filters)
    @objects = @ssh_keys

    render_index
  end

end
