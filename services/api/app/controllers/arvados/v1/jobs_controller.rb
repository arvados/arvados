class Arvados::V1::JobsController < ApplicationController
  accept_attribute_as_json :script_parameters, Hash
  accept_attribute_as_json :runtime_constraints, Hash
  accept_attribute_as_json :tasks_summary, Hash
  skip_before_filter :find_object_by_uuid, :only => :queue
  skip_before_filter :render_404_if_no_object, :only => :queue

  def index
    want_ancestor = @where[:script_version_descends_from]
    if want_ancestor
      # Check for missing commit_ancestor rows, and create them if
      # possible.
      @objects.
        dup.
        includes(:commit_ancestors). # I wish Rails would let me
                                     # specify here which
                                     # commit_ancestors I am
                                     # interested in.
        each do |o|
        if o.commit_ancestors.
            select { |ca| ca.ancestor == want_ancestor }.
            empty? and !o.script_version.nil?
          begin
            o.commit_ancestors << CommitAncestor.find_or_create_by_descendant_and_ancestor(o.script_version, want_ancestor)
          rescue
          end
        end
        o.commit_ancestors.
          select { |ca| ca.ancestor == want_ancestor }.
          select(&:is).
          first
      end
      # Now it is safe to do an .includes().where() because we are no
      # longer interested in jobs that have other ancestors but not
      # want_ancestor.
      @objects = @objects.
        includes(:commit_ancestors).
        where('commit_ancestors.ancestor = ? and commit_ancestors.is = ?',
              want_ancestor, true)
    end
    super
  end

  def cancel
    reload_object_before_update
    @object.update_attributes cancelled_at: Time.now
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
                    cancelled_at: nil
                  })
    params[:order] ||= 'priority desc, created_at'
    find_objects_for_index
    index
  end

  def self._queue_requires_parameters
    self._index_requires_parameters
  end
end
