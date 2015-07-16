class Arvados::V1::VirtualMachinesController < ApplicationController
  skip_before_filter :find_object_by_uuid, :only => :get_all_logins
  skip_before_filter :render_404_if_no_object, :only => :get_all_logins
  before_filter(:admin_required,
                :only => [:logins, :get_all_logins])

  def logins
    get_all_logins
  end

  def get_all_logins
    @response = []
    @vms = VirtualMachine.eager_load :login_permissions
    if @object
      @vms = @vms.where uuid: @object.uuid
    else
      @vms = @vms.all
    end
    @users = {}
    User.eager_load(:authorized_keys).
      where('users.uuid in (?)',
            @vms.map { |vm| vm.login_permissions.map &:tail_uuid }.flatten.uniq).
      each do |u|
      @users[u.uuid] = u
    end
    @vms.each do |vm|
      vm.login_permissions.each do |perm|
        user_uuid = perm.tail_uuid
        next if not @users[user_uuid]
        next if perm.properties['username'].blank?
        aks = @users[user_uuid].authorized_keys
        if aks.empty?
          # We'll emit one entry, with no public key.
          aks = [nil]
        end
        aks.each do |ak|
          @response << {
            username: perm.properties['username'],
            hostname: vm.hostname,
            groups: (perm.properties['groups'].to_a rescue []),
            public_key: ak ? ak.public_key : nil,
            user_uuid: user_uuid,
            virtual_machine_uuid: vm.uuid,
            authorized_key_uuid: ak ? ak.uuid : nil,
          }
        end
      end
    end
    send_json kind: "arvados#HashList", items: @response
  end
end
