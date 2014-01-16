class Arvados::V1::KeepDisksController < ApplicationController
  skip_before_filter :require_auth_scope_all, :only => :ping

  def self._ping_requires_parameters
    {
      uuid: false,
      ping_secret: true,
      node_uuid: false,
      filesystem_uuid: false,
      service_host: false,
      service_port: true,
      service_ssl_flag: true
    }
  end
  def ping
    if !@object
      if current_user.andand.is_admin
        @object = KeepDisk.new(filesystem_uuid: params[:filesystem_uuid])
        @object.save!

        # In the first ping from this new filesystem_uuid, we can't
        # expect the keep node to know the ping_secret so we made sure
        # we got an admin token. Here we add ping_secret to params so
        # KeepNode.ping() understands this update is properly
        # authenticated.
        params[:ping_secret] = @object.ping_secret
      else
        return render_not_found "object not found"
      end
    end

    params[:service_host] ||= request.env['REMOTE_ADDR']
    if not @object.ping params
      return render_not_found "object not found"
    end
    render json: @object.as_api_response(:superuser)
  end

  def find_objects_for_index
    # all users can list all keep disks
    @objects = model_class.where('1=1')
    super
  end
end
