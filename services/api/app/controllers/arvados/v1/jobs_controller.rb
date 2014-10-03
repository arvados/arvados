class Arvados::V1::JobsController < ApplicationController
  accept_attribute_as_json :script_parameters, Hash
  accept_attribute_as_json :runtime_constraints, Hash
  accept_attribute_as_json :tasks_summary, Hash
  skip_before_filter :find_object_by_uuid, :only => [:queue, :queue_size]
  skip_before_filter :render_404_if_no_object, :only => [:queue, :queue_size]

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

    if params[:find_or_create]
      return if false.equal?(load_filters_param)
      if @filters.empty?  # Translate older creation parameters into filters.
        @filters =
          [["repository", "=", resource_attrs[:repository]],
           ["script", "=", resource_attrs[:script]],
           ["script_version", "in git",
            params[:minimum_script_version] || resource_attrs[:script_version]],
           ["script_version", "not in git", params[:exclude_script_versions]],
          ].reject { |filter| filter.last.nil? or filter.last.empty? }
        if image_search = resource_attrs[:runtime_constraints].andand["docker_image"]
          if image_tag = resource_attrs[:runtime_constraints]["docker_image_tag"]
            image_search += ":#{image_tag}"
          end
          @filters.append(["docker_image_locator", "in docker", image_search])
        else
          @filters.append(["docker_image_locator", "=", nil])
        end
        begin
          load_job_specific_filters
        rescue ArgumentError => error
          return send_error(error.message)
        end
      end

      # Check specified filters for some reasonableness.
      filter_names = @filters.map { |f| f.first }.uniq
      ["repository", "script"].each do |req_filter|
        if not filter_names.include?(req_filter)
          return send_error("#{req_filter} filter required")
        end
      end

      # Search for a reusable Job, and return it if found.
      @objects = Job.readable_by(current_user)
      apply_filters
      @object = nil
      incomplete_job = nil
      @objects.each do |j|
        if j.nondeterministic != true and
            ["Queued", "Running", "Complete"].include?(j.state) and
            j.script_parameters == resource_attrs[:script_parameters]
          if j.state != "Complete" && j.owner_uuid == current_user.uuid
            # We'll use this if we don't find a job that has completed
            incomplete_job ||= j
          else
            if Collection.readable_by(current_user).find_by_portable_data_hash(j.output)
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
    @where.merge!({state: Job::Queued})
    return if false.equal?(load_filters_param)
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

  def load_job_specific_filters
    # Convert Job-specific @filters entries into general SQL filters.
    script_info = {"repository" => nil, "script" => nil}
    script_range = {"exclude_versions" => []}
    @filters.select! do |filter|
      if (script_info.has_key? filter[0]) and (filter[1] == "=")
        if script_info[filter[0]].nil?
          script_info[filter[0]] = filter[2]
        elsif script_info[filter[0]] != filter[2]
          raise ArgumentError.new("incompatible #{filter[0]} filters")
        end
      end
      case filter[0..1]
      when ["script_version", "in git"]
        script_range["min_version"] = filter.last
        false
      when ["script_version", "not in git"]
        begin
          script_range["exclude_versions"] += filter.last
        rescue TypeError
          script_range["exclude_versions"] << filter.last
        end
        false
      when ["docker_image_locator", "in docker"], ["docker_image_locator", "not in docker"]
        filter[1].sub!(/ docker$/, '')
        search_list = filter[2].is_a?(Enumerable) ? filter[2] : [filter[2]]
        filter[2] = search_list.flat_map do |search_term|
          image_search, image_tag = search_term.split(':', 2)
          Collection.find_all_for_docker_image(image_search, image_tag, @read_users).map(&:portable_data_hash)
        end
        true
      else
        true
      end
    end

    # Build a real script_version filter from any "not? in git" filters.
    if (script_range.size > 1) or script_range["exclude_versions"].any?
      script_info.each_pair do |key, value|
        if value.nil?
          raise ArgumentError.new("script_version filter needs #{key} filter")
        end
      end
      last_version = begin resource_attrs[:script_version] rescue "HEAD" end
      version_range = Commit.find_commit_range(current_user,
                                               script_info["repository"],
                                               script_range["min_version"],
                                               last_version,
                                               script_range["exclude_versions"])
      if version_range.nil?
        raise ArgumentError.
          new(["error searching #{script_info['repository']} from",
               "'#{script_range['min_version']}' to '#{last_version}',",
               "excluding #{script_range['exclude_versions']}"].join(" "))
      end
      @filters.append(["script_version", "in", version_range])
    end
  end

  def load_filters_param
    begin
      super
      load_job_specific_filters
    rescue ArgumentError => error
      send_error(error.message)
      false
    end
  end
end
