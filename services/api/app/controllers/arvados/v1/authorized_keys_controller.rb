class Arvados::V1::AuthorizedKeysController < ApplicationController
  before_filter :admin_required, :only => :get_all_logins
  def get_all_logins
    @users = {}
    User.includes(:authorized_keys).all.each do |u|
      @users[u.uuid] = u
    end
    @response = []
    @vms = VirtualMachine.includes(:login_permissions).all
    @vms.each do |vm|
      vm.login_permissions.each do |perm|
        user_uuid = perm.tail_uuid
        @users[user_uuid].andand.authorized_keys.each do |ak|
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
end
