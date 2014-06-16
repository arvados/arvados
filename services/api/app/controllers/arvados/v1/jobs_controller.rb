class Arvados::V1::JobsController < ApplicationController
  accept_attribute_as_json :script_parameters, Hash
  accept_attribute_as_json :runtime_constraints, Hash
  accept_attribute_as_json :tasks_summary, Hash
  skip_before_filter :find_object_by_uuid, :only => :queue
  skip_before_filter :render_404_if_no_object, :only => :queue

  def create
    [:repository, :script, :script_version, :script_parameters].each do |r|
      if !resource_attrs[r]
        return render json: {
          :errors => ["#{r} attribute must be specified"]
        }, status: :unprocessable_entity
      end
    end
    load_filters_param

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

    if params[:find_or_create]
      # Convert old special-purpose creation parameters to the new
      # filters-based method.
      minimum_script_version = params[:minimum_script_version]
      exclude_script_versions = params.fetch(:exclude_script_versions, [])
      @filters.select do |(col_name, operand, operator)|
        case col_name
        when "script_version"
          case operand
          when "in range"
            minimum_script_version = operator
            false
          when "not in", "not in range"
            begin
              exclude_script_versions += operator
            rescue TypeError
              exclude_script_versions << operator
            end
            false
          else
            true
          end
        else
          true
        end
      end
      @filters.append(["script_version", "in",
                       Commit.find_commit_range(current_user,
                                                resource_attrs[:repository],
                                                minimum_script_version,
                                                resource_attrs[:script_version],
                                                exclude_script_versions)])

      # Set up default filters for specific parameters.
      if @filters.select { |f| f.first == "script" }.empty?
        @filters.append(["script", "=", resource_attrs[:script]])
      end

      @objects = Job.readable_by(current_user)
      apply_filters
      @object = nil
      incomplete_job = nil
      @objects.each do |j|
        if j.nondeterministic != true and
            ((j.success == true and j.output != nil) or j.running == true) and
            j.script_parameters == resource_attrs[:script_parameters]
          if j.running
            # We'll use this if we don't find a job that has completed
            incomplete_job ||= j
          else
            # Record the first job in the list
            if !@object
              @object = j
            end
            # Ensure that all candidate jobs actually did produce the same output
            if @object.output != j.output
              @object = nil
              break
            end
          end
        end
        @object ||= incomplete_job
        if @object
          return show
        end
      end
    end

    super
  end

  def cancel
    reload_object_before_update
    @object.update_attributes! cancelled_at: Time.now
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
        last_ack_at ||= Time.now - Q_UPDATE_INTERVAL - 1
        if Time.now - last_ack_at >= Q_UPDATE_INTERVAL
          nodes_in_state = {idle: 0, alloc: 0}
          ActiveRecord::Base.uncached do
            Node.where('hostname is not ?', nil).collect do |n|
              if n.info[:slurm_state]
                nodes_in_state[n.info[:slurm_state]] ||= 0
                nodes_in_state[n.info[:slurm_state]] += 1
              end
            end
          end
          job_queue = Job.queue
          n_queued_before_me = 0
          job_queue.each do |j|
            break if j.uuid == @job.uuid
            n_queued_before_me += 1
          end
          yield "#{Time.now}" \
            " job #{@job.uuid}" \
            " queue_position #{n_queued_before_me}" \
            " queue_size #{job_queue.size}" \
            " nodes_idle #{nodes_in_state[:idle]}" \
            " nodes_alloc #{nodes_in_state[:alloc]}\n"
          last_ack_at = Time.now
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
    @where.merge!({
                    started_at: nil,
                    is_locked_by_uuid: nil,
                    cancelled_at: nil,
                    success: nil
                  })
    load_filters_param
    find_objects_for_index
    index
  end

  def self._queue_requires_parameters
    self._index_requires_parameters
  end
end
