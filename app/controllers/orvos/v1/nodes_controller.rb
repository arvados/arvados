class Orvos::V1::NodesController < ApplicationController
  def create
    @object = Node.new
    @object.save!
    @object.start!(lambda { |h| orvos_v1_ping_node_url(h) })
    show
  end

  def show
    render json: @object.to_json
  end

  def ping
    @object.ping({ ip: request.env['REMOTE_ADDR'],
                   ping_secret: params[:ping_secret],
                   ec2_instance_id: params[:ec2_instance_id] })
    show
  end
end
