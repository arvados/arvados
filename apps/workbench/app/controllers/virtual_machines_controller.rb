class VirtualMachinesController < ApplicationController
  def index
    @objects ||= model_class.all
    @vm_logins = {}
    if @objects.andand.first
      Link.where(tail_uuid: current_user.uuid,
                 head_uuid: @objects.collect(&:uuid),
                 link_class: 'permission',
                 name: 'can_login').
        each do |perm_link|
        if perm_link.properties.andand[:username]
          @vm_logins[perm_link.head_uuid] ||= []
          @vm_logins[perm_link.head_uuid] << perm_link.properties[:username]
        end
      end
      @objects.each do |vm|
        vm.current_user_logins = @vm_logins[vm.uuid].andand.compact || []
      end
    end
    super
  end
end
