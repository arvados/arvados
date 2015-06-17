class Arvados::V1::VirtualMachinesController < ApplicationController
  skip_before_filter :find_object_by_uuid, :only => :get_all_logins
  skip_before_filter :render_404_if_no_object, :only => :get_all_logins
  before_filter(:admin_required,
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
          unless perm.properties['username'].blank?
            @response << {
              username: perm.properties['username'],
              hostname: vm.hostname,
              groups: (perm.properties["groups"].to_a rescue []),
              public_key: ak.public_key,
              user_uuid: user_uuid,
              virtual_machine_uuid: vm.uuid,
              authorized_key_uuid: ak.uuid
            }
          end
        end
      end
    end
    send_json kind: "arvados#HashList", items: @response
  end
end
