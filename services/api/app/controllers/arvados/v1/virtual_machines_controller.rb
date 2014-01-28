class Arvados::V1::VirtualMachinesController < ApplicationController
  skip_before_filter :find_object_by_uuid, :only => :get_all_logins
  skip_before_filter :render_404_if_no_object, :only => :get_all_logins
  skip_before_filter(:require_auth_scope_all,
                     :only => [:logins, :get_all_logins])
  before_filter(:admin_required,
                :only => [:logins, :get_all_logins])
  before_filter(:require_auth_scope_for_get_all_logins,
                :only => [:logins, :get_all_logins])

  def logins
    get_all_logins
  end

  def get_all_logins
    @users = {}
    User.includes(:authorized_keys).all.each do |u|
      @users[u.uuid] = u
    end
    @response = []
    @vms = VirtualMachine.includes(:login_permissions)
    if @object
      @vms = @vms.where('uuid=?', @object.uuid)
    else
      @vms = @vms.all
    end
    @vms.each do |vm|
      vm.login_permissions.each do |perm|
        user_uuid = perm.tail_uuid
        @users[user_uuid].andand.authorized_keys.andand.each do |ak|
          username = perm.properties.andand['username']
          if username
            @response << {
              username: username,
              hostname: vm.hostname,
              public_key: ak.public_key,
              user_uuid: user_uuid,
              virtual_machine_uuid: vm.uuid,
              authorized_key_uuid: ak.uuid
            }
          end
        end
      end
    end
    render json: { kind: "arvados#HashList", items: @response }
  end

  protected

  def require_auth_scope_for_get_all_logins
    if @object
      # Client wants all logins for a single VM.
      require_auth_scope(['all', arvados_v1_virtual_machine_url(@object.uuid)])
    else
      # ...for a non-existent VM, or all VMs.
      require_auth_scope(['all'])
    end
  end
end
