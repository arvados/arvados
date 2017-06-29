# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class Arvados::V1::KeepDisksController < ApplicationController
  skip_before_filter :require_auth_scope, only: :ping
  skip_before_filter :render_404_if_no_object, only: :ping

  def self._ping_requires_parameters
    {
      uuid: {required: false},
      ping_secret: {required: true},
      node_uuid: {required: false},
      filesystem_uuid: {required: false},
      service_host: {required: false},
      service_port: {required: true},
      service_ssl_flag: {required: true}
    }
  end

  def ping
    params[:service_host] ||= request.env['REMOTE_ADDR']
    if !params[:uuid] && current_user.andand.is_admin
      # Create a new KeepDisk and ping it.
      @object = KeepDisk.new(filesystem_uuid: params[:filesystem_uuid])
      @object.save!

      # In the first ping from this new filesystem_uuid, we can't
      # expect the keep node to know the ping_secret so we made sure
      # we got an admin token. Here we add ping_secret to params so
      # the ping call below is properly authenticated.
      params[:ping_secret] = @object.ping_secret
    end
    act_as_system_user do
      if !@object.andand.ping(params)
        return render_not_found "object not found"
      end
      # Render the :superuser view (i.e., include the ping_secret) even
      # if !current_user.is_admin. This is safe because @object.ping's
      # success implies the ping_secret was already known by the client.
      send_json @object.as_api_response(:superuser)
    end
  end

  def find_objects_for_index
    # all users can list all keep disks
    @objects = model_class.where('1=1')
    super
  end
end
