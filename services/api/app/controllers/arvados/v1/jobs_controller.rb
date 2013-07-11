class Arvados::V1::JobsController < ApplicationController
  accept_attribute_as_json :script_parameters, Hash
  accept_attribute_as_json :resource_limits, Hash
  accept_attribute_as_json :tasks_summary, Hash
  skip_before_filter :find_object_by_uuid, :only => :queue

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

  class LogStreamer
    def initialize(job)
      @job = job
    end
    def each
      if @job.finished_at
        yield "#{@job.uuid} finished at #{@job.finished_at}\n"
        return
      end
      @redis = Redis.new(:timeout => 0)
      @redis.subscribe(@job.uuid) do |event|
        event.message do |channel, msg|
          if msg == "end"
            @redis.unsubscribe @job.uuid
          else
            yield msg
          end
        end
      end
    end
  end

  def log_tail_follow
    if !@object.andand.uuid
      return render_not_found
    end
    self.response.headers['Last-Modified'] = Time.now.ctime.to_s
    self.response_body = LogStreamer.new @object
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
