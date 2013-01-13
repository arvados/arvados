class Orvos::V1::NodesController < ApplicationController
  def create
    @object = Node.new
    @object.save!
    @object.start!(lambda { |h| orvos_v1_ping_node_url(h) })
    show
  end

  def ping
    @object.ping({ ip: params[:local_ipv4] || request.env['REMOTE_ADDR'],
                   ping_secret: params[:ping_secret],
                   ec2_instance_id: params[:instance_id] })
    show
  end
end
