class Arvados::V1::NodesController < ApplicationController
  skip_before_filter :require_auth_scope, :only => :ping
  skip_before_filter :find_object_by_uuid, :only => :ping
  skip_before_filter :render_404_if_no_object, :only => :ping

  def create
    @object = Node.new
    @object.save!
    @object.start!(lambda { |h| ping_arvados_v1_node_url(h) })
    show
  end

  def self._ping_requires_parameters
    { ping_secret: true }
  end

  def ping
    act_as_system_user do
      @object = Node.where(uuid: (params[:id] || params[:uuid])).first
      if !@object
        return render_not_found
      end
      ping_data = {
        ip: params[:local_ipv4] || request.env['REMOTE_ADDR'],
        ec2_instance_id: params[:instance_id]
      }
      [:ping_secret, :total_cpu_cores, :total_ram_mb, :total_scratch_mb]
        .each do |key|
        ping_data[key] = params[key] if params[key]
      end
      @object.ping(ping_data)
      if @object.info['ping_secret'] == params[:ping_secret]
        render json: @object.as_api_response(:superuser)
      else
        raise "Invalid ping_secret after ping"
      end
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
