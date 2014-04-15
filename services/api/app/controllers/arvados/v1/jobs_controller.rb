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
          :error => "#{r} attribute must be specified"
        }, status: :unprocessable_entity
      end
    end

    r = Commit.find_commit_range(current_user,
                                 resource_attrs[:repository],
                                 resource_attrs[:minimum_script_version],
                                 resource_attrs[:script_version],
                                 resource_attrs[:exclude_script_versions])
    if !resource_attrs[:nondeterministic] and !resource_attrs[:no_reuse]
      # Search for jobs whose script_version is in the list of commits
      # returned by find_commit_range
      @object = nil
      @incomplete_job = nil
      Job.readable_by(current_user).where(script: resource_attrs[:script],
                                          script_version: r).
        each do |j|
        if j.nondeterministic != true and
            ((j.success == true and j.output != nil) or j.running == true) and
            j.script_parameters == resource_attrs[:script_parameters]
          if j.running
            # We'll use this if we don't find a job that has completed
            @incomplete_job ||= j
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
        @object ||= @incomplete_job
        if @object
          return show
        end
      end
    end
    if r
      resource_attrs[:script_version] = r[0]
    end

    # Don't pass these on to activerecord
    resource_attrs.delete(:minimum_script_version)
    resource_attrs.delete(:exclude_script_versions)
    resource_attrs.delete(:no_reuse)
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
      @redis = Redis.new(:timeout => 0)
      if @redis.exists @job.uuid
        # A log buffer exists. Start by showing the last few KB.
        @redis.
          getrange(@job.uuid, 0 - [@opts[:buffer_size], 1].max, -1).
          sub(/^[^\n]*\n?/, '').
          split("\n").
          each do |line|
          yield "#{line}\n"
        end
      end
      # TODO: avoid missing log entries between getrange() above and
      # subscribe() below.
      @redis.subscribe(@job.uuid) do |event|
        event.message do |channel, msg|
          if msg == "end"
            @redis.unsubscribe @job.uuid
          else
            yield "#{msg}\n"
          end
        end
      end
    end
  end

  def self._log_tail_follow_requires_parameters
    {
      buffer_size: {type: 'integer', required: false, default: 2**13}
    }
  end
  def log_tail_follow
    if !@object.andand.uuid
      return render_not_found
    end
    if client_accepts_plain_text_stream
      self.response.headers['Last-Modified'] = Time.now.ctime.to_s
      self.response_body = LogStreamer.new @object, {
        buffer_size: (params[:buffer_size].to_i rescue 2**13)
      }
    else
      render json: {
        href: url_for(uuid: @object.uuid),
        comment: ('To retrieve the log stream as plain text, ' +
                  'use a request header like "Accept: text/plain"')
      }
    end
  end

  def queue
    load_where_param
    @where.merge!({
                    started_at: nil,
                    is_locked_by_uuid: nil,
                    cancelled_at: nil,
                    success: nil
                  })
    params[:order] ||= 'priority desc, created_at'
    find_objects_for_index
    index
  end

  def self._queue_requires_parameters
    self._index_requires_parameters
  end
end
