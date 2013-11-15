class Arvados::V1::KeepDisksController < ApplicationController
  skip_before_filter :require_auth_scope_all, :only => :ping

  def self._ping_requires_parameters
    {
      uuid: false,
      ping_secret: true,
      ec2_instance_id: false,
      local_ipv4: false,
      filesystem_uuid: false,
      service_port: true,
      service_ssl_flag: true
    }
  end
  def ping
    if !@object and params[:filesystem_uuid] and current_user and current_user.is_admin
      if KeepDisk.where('filesystem_uuid=?', params[:filesystem_uuid]).empty?
        @object = KeepDisk.new filesystem_uuid: params[:filesystem_uuid]
        @object.save!
        params[:ping_secret] = @object.ping_secret
      else
        raise "ping from keep_disk with existing filesystem_uuid #{params[:filesystem_uuid]} but wrong uuid #{params[:uuid]}"
      end
    end

    if !@object
      return render_not_found "object not found"
    end

    params.merge!(service_host:
                  params[:local_ipv4] || request.env['REMOTE_ADDR'])
    @object.ping params
    show
  end

  def find_objects_for_index
    if current_user.andand.is_admin || !current_user.andand.is_active
      super
    else
      # active non-admin users can list all keep disks
      @objects = model_class.all
    end
  end
end
