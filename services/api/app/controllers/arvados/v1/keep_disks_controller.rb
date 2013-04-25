class Arvados::V1::KeepDisksController < ApplicationController
  skip_before_filter :login_required, :only => :ping

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
end
