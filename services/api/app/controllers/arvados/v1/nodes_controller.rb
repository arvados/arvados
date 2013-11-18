class Arvados::V1::NodesController < ApplicationController
  skip_before_filter :require_auth_scope_all, :only => :ping

  def create
    @object = Node.new
    @object.save!
    @object.start!(lambda { |h| arvados_v1_ping_node_url(h) })
    show
  end

  def self._ping_requires_parameters
    { ping_secret: true }
  end
  def ping
    @object.ping({ ip: params[:local_ipv4] || request.env['REMOTE_ADDR'],
                   ping_secret: params[:ping_secret],
                   ec2_instance_id: params[:instance_id] })
    if @object.info[:ping_secret] == params[:ping_secret]
      render json: @object.as_api_response(:superuser)
    else
      raise "Invalid ping_secret after ping"
    end
  end

  def find_objects_for_index
    if current_user.andand.is_admin || !current_user.andand.is_active
      super
    else
      # active non-admin users can list nodes that are (or were
      # recently) working
      @objects = model_class.where('last_ping_at >= ?', Time.now - 1.hours)
    end
  end
end
