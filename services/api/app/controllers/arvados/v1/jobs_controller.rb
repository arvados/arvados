# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class Arvados::V1::JobsController < ApplicationController
  accept_attribute_as_json :components, Hash
  accept_attribute_as_json :script_parameters, Hash
  accept_attribute_as_json :runtime_constraints, Hash
  accept_attribute_as_json :tasks_summary, Hash
  skip_before_filter :find_object_by_uuid, :only => [:queue, :queue_size]
  skip_before_filter :render_404_if_no_object, :only => [:queue, :queue_size]

  include DbCurrentTime

  def create
    [:repository, :script, :script_version, :script_parameters].each do |r|
      if !resource_attrs[r]
        return send_error("#{r} attribute must be specified",
                          status: :unprocessable_entity)
      end
    end

    # We used to ask for the minimum_, exclude_, and no_reuse params
    # in the job resource. Now we advertise them as flags that alter
    # the behavior of the create action.
    [:minimum_script_version, :exclude_script_versions].each do |attr|
      if resource_attrs.has_key? attr
        params[attr] = resource_attrs.delete attr
      end
    end
    if resource_attrs.has_key? :no_reuse
      params[:find_or_create] = !resource_attrs.delete(:no_reuse)
    end

    return super if !params[:find_or_create]
    return if !load_filters_param

    begin
      @object = Job.find_reusable(resource_attrs, params, @filters, @read_users)
    rescue ArgumentError => error
      return send_error(error.message)
    end

    if @object
      show
    else
      super
    end
  end

  def cancel
    reload_object_before_update
    @object.cancel cascade: params[:cascade]
    show
  end

  def lock
    @object.lock current_user.uuid
    show
  end

  class LogStreamer
    Q_UPDATE_INTERVAL = 12
    def initialize(job, opts={})
      @job = job
      @opts = opts
    end
    def each
      if @job.finished_at
        yield "#{@job.uuid} finished at #{@job.finished_at}\n"
        return
      end
      while not @job.started_at
        # send a summary (job queue + available nodes) to the client
        # every few seconds while waiting for the job to start
        current_time = db_current_time
        last_ack_at ||= current_time - Q_UPDATE_INTERVAL - 1
        if current_time - last_ack_at >= Q_UPDATE_INTERVAL
          nodes_in_state = {idle: 0, alloc: 0}
          ActiveRecord::Base.uncached do
            Node.where('hostname is not ?', nil).collect do |n|
              if n.info[:slurm_state]
                nodes_in_state[n.info[:slurm_state]] ||= 0
                nodes_in_state[n.info[:slurm_state]] += 1
              end
            end
          end
          job_queue = Job.queue.select(:uuid)
          n_queued_before_me = 0
          job_queue.each do |j|
            break if j.uuid == @job.uuid
            n_queued_before_me += 1
          end
          yield "#{db_current_time}" \
            " job #{@job.uuid}" \
            " queue_position #{n_queued_before_me}" \
            " queue_size #{job_queue.count}" \
            " nodes_idle #{nodes_in_state[:idle]}" \
            " nodes_alloc #{nodes_in_state[:alloc]}\n"
          last_ack_at = db_current_time
        end
        sleep 3
        ActiveRecord::Base.uncached do
          @job.reload
        end
      end
    end
  end

  def queue
    params[:order] ||= ['priority desc', 'created_at']
    load_limit_offset_order_params
    load_where_param
    @where.merge!({state: Job::Queued})
    return if !load_filters_param
    find_objects_for_index
    index
  end

  def queue_size
    # Users may not be allowed to see all the jobs in the queue, so provide a
    # method to get just the queue size in order to get a gist of how busy the
    # cluster is.
    render :json => {:queue_size => Job.queue.size}
  end

  def self._create_requires_parameters
    (super rescue {}).
      merge({
              find_or_create: {
                type: 'boolean', required: false, default: false
              },
              filters: {
                type: 'array', required: false
              },
              minimum_script_version: {
                type: 'string', required: false
              },
              exclude_script_versions: {
                type: 'array', required: false
              },
            })
  end

  def self._queue_requires_parameters
    self._index_requires_parameters
  end

  protected

  def load_filters_param
    begin
      super
      attrs = resource_attrs rescue {}
      @filters = Job.load_job_specific_filters attrs, @filters, @read_users
    rescue ArgumentError => error
      send_error(error.message)
      false
    else
      true
    end
  end
end
