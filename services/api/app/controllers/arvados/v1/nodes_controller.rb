# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class Arvados::V1::NodesController < ApplicationController
  skip_before_action :require_auth_scope, :only => :ping
  skip_before_action :find_object_by_uuid, :only => :ping
  skip_before_action :render_404_if_no_object, :only => :ping

  include DbCurrentTime

  def self._ping_requires_parameters
    { ping_secret: {required: true} }
  end

  def self._create_requires_parameters
    super.merge(
      { assign_slot: {required: false, type: 'boolean', description: 'assign slot and hostname'} })
  end

  def self._update_requires_parameters
    super.merge(
      { assign_slot: {required: false, type: 'boolean', description: 'assign slot and hostname'} })
  end

  def create
    @object = model_class.new(resource_attrs)
    @object.assign_slot if params[:assign_slot]
    @object.save!
    show
  end

  def update
    if resource_attrs[:job_uuid].is_a? String
      @object.job_readable = readable_job_uuids([resource_attrs[:job_uuid]]).any?
    end
    attrs_to_update = resource_attrs.reject { |k,v|
      [:kind, :etag, :href].index k
    }
    @object.update!(attrs_to_update)
    @object.assign_slot if params[:assign_slot]
    @object.save!
    show
  end

  def ping
    act_as_system_user do
      @object = Node.where(uuid: (params[:id] || params[:uuid])).first
      if !@object
        return render_not_found
      end
      ping_data = {
        ip: params[:local_ipv4] || request.remote_ip,
        ec2_instance_id: params[:instance_id]
      }
      [:ping_secret, :total_cpu_cores, :total_ram_mb, :total_scratch_mb]
        .each do |key|
        ping_data[key] = params[key] if params[key]
      end
      @object.ping(ping_data)
      if @object.info['ping_secret'] == params[:ping_secret]
        send_json @object.as_api_response(:superuser)
      else
        raise "Invalid ping_secret after ping"
      end
    end
  end

  def find_objects_for_index
    if !current_user.andand.is_admin && current_user.andand.is_active
      # active non-admin users can list nodes that are (or were
      # recently) working
      @objects = model_class.where('last_ping_at >= ?', db_current_time - 1.hours)
    end
    super
    if @select.nil? or @select.include? 'job_uuid'
      job_uuids = @objects.map { |n| n[:job_uuid] }.compact
      assoc_jobs = readable_job_uuids(job_uuids)
      @objects.each do |node|
        node.job_readable = assoc_jobs.include?(node[:job_uuid])
      end
    end
  end

  protected

  def readable_job_uuids(uuids)
    Job.readable_by(*@read_users).select(:uuid).where(uuid: uuids).map(&:uuid)
  end
end
