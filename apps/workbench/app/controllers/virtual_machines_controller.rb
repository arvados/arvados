# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class VirtualMachinesController < ApplicationController
  def index
    @objects ||= model_class.all
    @vm_logins = {}
    if @objects.andand.first
      Link.where(tail_uuid: current_user.uuid,
                 head_uuid: @objects.collect(&:uuid),
                 link_class: 'permission',
                 name: 'can_login').with_count("none").
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

  def webshell
    return render_not_found if Rails.configuration.Services.WebShell.ExternalURL == URI("")
    webshell_url = URI(Rails.configuration.Services.WebShell.ExternalURL)
    if webshell_url.host.index("*") != nil
      webshell_url.host = webshell_url.host.sub("*", @object.hostname)
    else
      webshell_url.path = "/#{@object.hostname}"
    end
    @webshell_url = webshell_url.to_s
    render layout: false
  end

end
