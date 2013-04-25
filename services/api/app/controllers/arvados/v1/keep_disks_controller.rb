class Arvados::V1::KeepDisksController < ApplicationController
  skip_before_filter :login_required, :only => :ping

  def self._ping_requires_parameters
    { ping_secret: true, uuid: false }
  end
  def ping
    @object.ping({ ip: params[:local_ipv4] || request.env['REMOTE_ADDR'],
                   ping_secret: params[:ping_secret],
                   ec2_instance_id: params[:instance_id] })
    show
  end
end
