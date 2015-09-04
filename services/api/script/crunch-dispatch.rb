#!/usr/bin/env ruby

require 'shellwords'
include Process

$options = {}
(ARGV.any? ? ARGV : ['--jobs', '--pipelines']).each do |arg|
  case arg
  when '--jobs'
    $options[:jobs] = true
  when '--pipelines'
    $options[:pipelines] = true
  else
    abort "Unrecognized command line option '#{arg}'"
  end
end
if not ($options[:jobs] or $options[:pipelines])
  abort "Nothing to do. Please specify at least one of: --jobs, --pipelines."
end

ARGV.reject! { |a| a =~ /--jobs|--pipelines/ }

$warned = {}
$signal = {}
%w{TERM INT}.each do |sig|
  signame = sig
  Signal.trap(sig) do
    $stderr.puts "Received #{signame} signal"
    $signal[:term] = true
  end
end

if ENV["CRUNCH_DISPATCH_LOCKFILE"]
  lockfilename = ENV.delete "CRUNCH_DISPATCH_LOCKFILE"
  lockfile = File.open(lockfilename, File::RDWR|File::CREAT, 0644)
  unless lockfile.flock File::LOCK_EX|File::LOCK_NB
    abort "Lock unavailable on #{lockfilename} - exit"
  end
end

ENV["RAILS_ENV"] = ARGV[0] || ENV["RAILS_ENV"] || "development"

require File.dirname(__FILE__) + '/../config/boot'
require File.dirname(__FILE__) + '/../config/environment'
require 'open3'

class LogTime < Time
  def to_s
    self.utc.strftime "%Y-%m-%d_%H:%M:%S"
  end
end

class Dispatcher
  include ApplicationHelper

  EXIT_TEMPFAIL = 75
  EXIT_RETRY_UNLOCKED = 93
  RETRY_UNLOCKED_LIMIT = 3

  def initialize
    @crunch_job_bin = (ENV['CRUNCH_JOB_BIN'] || `which arv-crunch-job`.strip)
    if @crunch_job_bin.empty?
      raise "No CRUNCH_JOB_BIN env var, and crunch-job not in path."
    end

    @docker_bin = ENV['CRUNCH_JOB_DOCKER_BIN']

    @arvados_internal = Rails.configuration.git_internal_dir
    if not File.exists? @arvados_internal
      $stderr.puts `mkdir -p #{@arvados_internal.shellescape} && git init --bare #{@arvados_internal.shellescape}`
      raise "No internal git repository available" unless ($? == 0)
    end

    @repo_root = Rails.configuration.git_repositories_dir
    @arvados_repo_path = Repository.where(name: "arvados").first.server_path
    @authorizations = {}
    @did_recently = {}
    @fetched_commits = {}
    @git_tags = {}
    @node_state = {}
    @pipe_auth_tokens = {}
    @running = {}
    @todo = []
    @todo_job_retries = {}
    @job_retry_counts = Hash.new(0)
    @todo_pipelines = []
  end

  def sysuser
    return act_as_system_user
  end

  def refresh_todo
    if $options[:jobs]
      @todo = @todo_job_retries.values + Job.queue.select(&:repository)
    end
    if $options[:pipelines]
      @todo_pipelines = PipelineInstance.queue
    end
  end

  def each_slurm_line(cmd, outfmt, max_fields=nil)
    max_fields ||= outfmt.split(":").size
    max_fields += 1  # To accommodate the node field we add
    @@slurm_version ||= Gem::Version.new(`sinfo --version`.match(/\b[\d\.]+\b/)[0])
    if Gem::Version.new('2.3') <= @@slurm_version
      `#{cmd} --noheader -o '%n:#{outfmt}'`.each_line do |line|
        yield line.chomp.split(":", max_fields)
      end
    else
      # Expand rows with hostname ranges (like "foo[1-3,5,9-12]:idle")
      # into multiple rows with one hostname each.
      `#{cmd} --noheader -o '%N:#{outfmt}'`.each_line do |line|
        tokens = line.chomp.split(":", max_fields)
        if (re = tokens[0].match /^(.*?)\[([-,\d]+)\]$/)
          tokens.shift
          re[2].split(",").each do |range|
            range = range.split("-").collect(&:to_i)
            (range[0]..range[-1]).each do |n|
              yield [re[1] + n.to_s] + tokens
            end
          end
        else
          yield tokens
        end
      end
    end
  end

  def slurm_status
    slurm_nodes = {}
    each_slurm_line("sinfo", "%t") do |hostname, state|
      # Treat nodes in idle* state as down, because the * means that slurm
      # hasn't been able to communicate with it recently.
      state.sub!(/^idle\*/, "down")
      state.sub!(/\W+$/, "")
      state = "down" unless %w(idle alloc down).include?(state)
      slurm_nodes[hostname] = {state: state, job: nil}
    end
    each_slurm_line("squeue", "%j") do |hostname, job_uuid|
      slurm_nodes[hostname][:job] = job_uuid if slurm_nodes[hostname]
    end
    slurm_nodes
  end

  def update_node_status
    return unless Server::Application.config.crunch_job_wrapper.to_s.match /^slurm/
    slurm_status.each_pair do |hostname, slurmdata|
      next if @node_state[hostname] == slurmdata
      begin
        node = Node.where('hostname=?', hostname).order(:last_ping_at).last
        if node
          $stderr.puts "dispatch: update #{hostname} state to #{slurmdata}"
          node.info["slurm_state"] = slurmdata[:state]
          node.job_uuid = slurmdata[:job]
          if node.save
            @node_state[hostname] = slurmdata
          else
            $stderr.puts "dispatch: failed to update #{node.uuid}: #{node.errors.messages}"
          end
        elsif slurmdata[:state] != 'down'
          $stderr.puts "dispatch: SLURM reports '#{hostname}' is not down, but no node has that name"
        end
      rescue => error
        $stderr.puts "dispatch: error updating #{hostname} node status: #{error}"
      end
    end
  end

  def positive_int(raw_value, default=nil)
    value = begin raw_value.to_i rescue 0 end
    if value > 0
      value
    else
      default
    end
  end

  NODE_CONSTRAINT_MAP = {
    # Map Job runtime_constraints keys to the corresponding Node info key.
    'min_ram_mb_per_node' => 'total_ram_mb',
    'min_scratch_mb_per_node' => 'total_scratch_mb',
    'min_cores_per_node' => 'total_cpu_cores',
  }

  def nodes_available_for_job_now(job)
    # Find Nodes that satisfy a Job's runtime constraints (by building
    # a list of Procs and using them to test each Node).  If there
    # enough to run the Job, return an array of their names.
    # Otherwise, return nil.
    need_procs = NODE_CONSTRAINT_MAP.each_pair.map do |job_key, node_key|
      Proc.new do |node|
        positive_int(node.info[node_key], 0) >=
          positive_int(job.runtime_constraints[job_key], 0)
      end
    end
    min_node_count = positive_int(job.runtime_constraints['min_nodes'], 1)
    usable_nodes = []
    Node.find_each do |node|
      good_node = (node.info['slurm_state'] == 'idle')
      need_procs.each { |node_test| good_node &&= node_test.call(node) }
      if good_node
        usable_nodes << node
        if usable_nodes.count >= min_node_count
          return usable_nodes.map { |node| node.hostname }
        end
      end
    end
    nil
  end

  def nodes_available_for_job(job)
    # Check if there are enough idle nodes with the Job's minimum
    # hardware requirements to run it.  If so, return an array of
    # their names.  If not, up to once per hour, signal start_jobs to
    # hold off launching Jobs.  This delay is meant to give the Node
    # Manager an opportunity to make new resources available for new
    # Jobs.
    #
    # The exact timing parameters here might need to be adjusted for
    # the best balance between helping the longest-waiting Jobs run,
    # and making efficient use of immediately available resources.
    # These are all just first efforts until we have more data to work
    # with.
    nodelist = nodes_available_for_job_now(job)
    if nodelist.nil? and not did_recently(:wait_for_available_nodes, 3600)
      $stderr.puts "dispatch: waiting for nodes for #{job.uuid}"
      @node_wait_deadline = Time.now + 5.minutes
    end
    nodelist
  end

  def fail_job job, message
    $stderr.puts "dispatch: #{job.uuid}: #{message}"
    begin
      Log.new(object_uuid: job.uuid,
              event_type: 'dispatch',
              owner_uuid: job.owner_uuid,
              summary: message,
              properties: {"text" => message}).save!
    rescue
      $stderr.puts "dispatch: log.create failed"
    end

    begin
      job.lock @authorizations[job.uuid].user.uuid
      job.state = "Failed"
      if not job.save
        $stderr.puts "dispatch: save failed setting job #{job.uuid} to failed"
      end
    rescue ArvadosModel::AlreadyLockedError
      $stderr.puts "dispatch: tried to mark job #{job.uuid} as failed but it was already locked by someone else"
    end
  end

  def stdout_s(cmd_a, opts={})
    IO.popen(cmd_a, "r", opts) do |pipe|
      return pipe.read.chomp
    end
  end

  def git_cmd(*cmd_a)
    ["git", "--git-dir=#{@arvados_internal}"] + cmd_a
  end

  def get_authorization(job)
    if @authorizations[job.uuid] and
        @authorizations[job.uuid].user.uuid != job.modified_by_user_uuid
      # We already made a token for this job, but we need a new one
      # because modified_by_user_uuid has changed (the job will run
      # as a different user).
      @authorizations[job.uuid].update_attributes expires_at: Time.now
      @authorizations[job.uuid] = nil
    end
    if not @authorizations[job.uuid]
      auth = ApiClientAuthorization.
        new(user: User.where('uuid=?', job.modified_by_user_uuid).first,
            api_client_id: 0)
      if not auth.save
        $stderr.puts "dispatch: auth.save failed for #{job.uuid}"
      else
        @authorizations[job.uuid] = auth
      end
    end
    @authorizations[job.uuid]
  end

  def internal_repo_has_commit? sha1
    if (not @fetched_commits[sha1] and
        sha1 == stdout_s(git_cmd("rev-list", "-n1", sha1), err: "/dev/null") and
        $? == 0)
      @fetched_commits[sha1] = true
    end
    return @fetched_commits[sha1]
  end

  def get_commit src_repo, sha1
    return true if internal_repo_has_commit? sha1

    # commit does not exist in internal repository, so import the
    # source repository using git fetch-pack
    cmd = git_cmd("fetch-pack", "--no-progress", "--all", src_repo)
    $stderr.puts "dispatch: #{cmd}"
    $stderr.puts(stdout_s(cmd))
    @fetched_commits[sha1] = ($? == 0)
  end

  def tag_commit(commit_hash, tag_name)
    # @git_tags[T]==V if we know commit V has been tagged T in the
    # arvados_internal repository.
    if not @git_tags[tag_name]
      cmd = git_cmd("tag", tag_name, commit_hash)
      $stderr.puts "dispatch: #{cmd}"
      $stderr.puts(stdout_s(cmd, err: "/dev/null"))
      unless $? == 0
        # git tag failed.  This may be because the tag already exists, so check for that.
        tag_rev = stdout_s(git_cmd("rev-list", "-n1", tag_name))
        if $? == 0
          # We got a revision back
          if tag_rev != commit_hash
            # Uh oh, the tag doesn't point to the revision we were expecting.
            # Someone has been monkeying with the job record and/or git.
            fail_job job, "Existing tag #{tag_name} points to commit #{tag_rev} but expected commit #{commit_hash}"
            return nil
          end
          # we're okay (fall through to setting @git_tags below)
        else
          # git rev-list failed for some reason.
          fail_job job, "'git tag' for #{tag_name} failed but did not find any existing tag using 'git rev-list'"
          return nil
        end
      end
      # 'git tag' was successful, or there is an existing tag that points to the same revision.
      @git_tags[tag_name] = commit_hash
    elsif @git_tags[tag_name] != commit_hash
      fail_job job, "Existing tag #{tag_name} points to commit #{@git_tags[tag_name]} but this job uses commit #{commit_hash}"
      return nil
    end
    @git_tags[tag_name]
  end

  def start_jobs
    @todo.each do |job|
      next if @running[job.uuid]

      cmd_args = nil
      case Server::Application.config.crunch_job_wrapper
      when :none
        if @running.size > 0
            # Don't run more than one at a time.
            return
        end
        cmd_args = []
      when :slurm_immediate
        nodelist = nodes_available_for_job(job)
        if nodelist.nil?
          if Time.now < @node_wait_deadline
            break
          else
            next
          end
        end
        cmd_args = ["salloc",
                    "--chdir=/",
                    "--immediate",
                    "--exclusive",
                    "--no-kill",
                    "--job-name=#{job.uuid}",
                    "--nodelist=#{nodelist.join(',')}"]
      else
        raise "Unknown crunch_job_wrapper: #{Server::Application.config.crunch_job_wrapper}"
      end

      if Server::Application.config.crunch_job_user
        cmd_args.unshift("sudo", "-E", "-u",
                         Server::Application.config.crunch_job_user,
                         "LD_LIBRARY_PATH=#{ENV['LD_LIBRARY_PATH']}",
                         "PATH=#{ENV['PATH']}",
                         "PERLLIB=#{ENV['PERLLIB']}",
                         "PYTHONPATH=#{ENV['PYTHONPATH']}",
                         "RUBYLIB=#{ENV['RUBYLIB']}",
                         "GEM_PATH=#{ENV['GEM_PATH']}")
      end

      next unless get_authorization job

      ready = internal_repo_has_commit? job.script_version

      if not ready
        # Import the commit from the specified repository into the
        # internal repository. This should have been done already when
        # the job was created/updated; this code is obsolete except to
        # avoid deployment races. Failing the job would be a
        # reasonable thing to do at this point.
        repo = Repository.where(name: job.repository).first
        if repo.nil? or repo.server_path.nil?
          fail_job "Repository #{job.repository} not found under #{@repo_root}"
          next
        end
        ready &&= get_commit repo.server_path, job.script_version
        ready &&= tag_commit job.script_version, job.uuid
      end

      # This should be unnecessary, because API server does it during
      # job create/update, but it's still not a bad idea to verify the
      # tag is correct before starting the job:
      ready &&= tag_commit job.script_version, job.uuid

      # The arvados_sdk_version doesn't support use of arbitrary
      # remote URLs, so the requested version isn't necessarily copied
      # into the internal repository yet.
      if job.arvados_sdk_version
        ready &&= get_commit @arvados_repo_path, job.arvados_sdk_version
        ready &&= tag_commit job.arvados_sdk_version, "#{job.uuid}-arvados-sdk"
      end

      if not ready
        fail_job job, "commit not present in internal repository"
        next
      end

      cmd_args += [@crunch_job_bin,
                   '--job-api-token', @authorizations[job.uuid].api_token,
                   '--job', job.uuid,
                   '--git-dir', @arvados_internal]

      if @docker_bin
        cmd_args += ['--docker-bin', @docker_bin]
      end

      if @todo_job_retries.include?(job.uuid)
        cmd_args << "--force-unlock"
      end

      $stderr.puts "dispatch: #{cmd_args.join ' '}"

      begin
        i, o, e, t = Open3.popen3(*cmd_args)
      rescue
        $stderr.puts "dispatch: popen3: #{$!}"
        sleep 1
        next
      end

      $stderr.puts "dispatch: job #{job.uuid}"
      start_banner = "dispatch: child #{t.pid} start #{LogTime.now}"
      $stderr.puts start_banner

      @running[job.uuid] = {
        stdin: i,
        stdout: o,
        stderr: e,
        wait_thr: t,
        job: job,
        buf: {stderr: '', stdout: ''},
        started: false,
        sent_int: 0,
        job_auth: @authorizations[job.uuid],
        stderr_buf_to_flush: '',
        stderr_flushed_at: Time.new(0),
        bytes_logged: 0,
        events_logged: 0,
        log_throttle_is_open: true,
        log_throttle_reset_time: Time.now + Rails.configuration.crunch_log_throttle_period,
        log_throttle_bytes_so_far: 0,
        log_throttle_lines_so_far: 0,
        log_throttle_bytes_skipped: 0,
      }
      i.close
      @todo_job_retries.delete(job.uuid)
      update_node_status
    end
  end

  # Test for hard cap on total output and for log throttling.  Returns whether
  # the log line should go to output or not.  Modifies "line" in place to
  # replace it with an error if a logging limit is tripped.
  def rate_limit running_job, line
    message = false
    linesize = line.size
    if running_job[:log_throttle_is_open]
      running_job[:log_throttle_lines_so_far] += 1
      running_job[:log_throttle_bytes_so_far] += linesize
      running_job[:bytes_logged] += linesize

      if (running_job[:bytes_logged] >
          Rails.configuration.crunch_limit_log_bytes_per_job)
        message = "Exceeded log limit #{Rails.configuration.crunch_limit_log_bytes_per_job} bytes (crunch_limit_log_bytes_per_job). Log will be truncated."
        running_job[:log_throttle_reset_time] = Time.now + 100.years
        running_job[:log_throttle_is_open] = false

      elsif (running_job[:log_throttle_bytes_so_far] >
             Rails.configuration.crunch_log_throttle_bytes)
        remaining_time = running_job[:log_throttle_reset_time] - Time.now
        message = "Exceeded rate #{Rails.configuration.crunch_log_throttle_bytes} bytes per #{Rails.configuration.crunch_log_throttle_period} seconds (crunch_log_throttle_bytes). Logging will be silenced for the next #{remaining_time.round} seconds.\n"
        running_job[:log_throttle_is_open] = false

      elsif (running_job[:log_throttle_lines_so_far] >
             Rails.configuration.crunch_log_throttle_lines)
        remaining_time = running_job[:log_throttle_reset_time] - Time.now
        message = "Exceeded rate #{Rails.configuration.crunch_log_throttle_lines} lines per #{Rails.configuration.crunch_log_throttle_period} seconds (crunch_log_throttle_lines), logging will be silenced for the next #{remaining_time.round} seconds.\n"
        running_job[:log_throttle_is_open] = false
      end
    end

    if not running_job[:log_throttle_is_open]
      # Don't log anything if any limit has been exceeded. Just count lossage.
      running_job[:log_throttle_bytes_skipped] += linesize
    end

    if message
      # Yes, write to logs, but use our "rate exceeded" message
      # instead of the log message that exceeded the limit.
      line.replace message
      true
    else
      running_job[:log_throttle_is_open]
    end
  end

  def read_pipes
    @running.each do |job_uuid, j|
      job = j[:job]

      now = Time.now
      if now > j[:log_throttle_reset_time]
        # It has been more than throttle_period seconds since the last
        # checkpoint so reset the throttle
        if j[:log_throttle_bytes_skipped] > 0
          message = "#{job_uuid} ! Skipped #{j[:log_throttle_bytes_skipped]} bytes of log"
          $stderr.puts message
          j[:stderr_buf_to_flush] << "#{LogTime.now} #{message}\n"
        end

        j[:log_throttle_reset_time] = now + Rails.configuration.crunch_log_throttle_period
        j[:log_throttle_bytes_so_far] = 0
        j[:log_throttle_lines_so_far] = 0
        j[:log_throttle_bytes_skipped] = 0
        j[:log_throttle_is_open] = true
      end

      j[:buf].each do |stream, streambuf|
        # Read some data from the child stream
        buf = ''
        begin
          # It's important to use a big enough buffer here. When we're
          # being flooded with logs, we must read and discard many
          # bytes at once. Otherwise, we can easily peg a CPU with
          # time-checking and other loop overhead. (Quick tests show a
          # 1MiB buffer working 2.5x as fast as a 64 KiB buffer.)
          #
          # So don't reduce this buffer size!
          buf = j[stream].read_nonblock(2**20)
        rescue Errno::EAGAIN, EOFError
        end

        # Short circuit the counting code if we're just going to throw
        # away the data anyway.
        if not j[:log_throttle_is_open]
          j[:log_throttle_bytes_skipped] += streambuf.size + buf.size
          streambuf.replace ''
          next
        elsif buf == ''
          next
        end

        # Append to incomplete line from previous read, if any
        streambuf << buf

        bufend = ''
        streambuf.each_line do |line|
          if not line.end_with? $/
            if line.size > Rails.configuration.crunch_log_throttle_bytes
              # Without a limit here, we'll use 2x an arbitrary amount
              # of memory, and waste a lot of time copying strings
              # around, all without providing any feedback to anyone
              # about what's going on _or_ hitting any of our throttle
              # limits.
              #
              # Here we leave "line" alone, knowing it will never be
              # sent anywhere: rate_limit() will reach
              # crunch_log_throttle_bytes immediately. However, we'll
              # leave [...] in bufend: if the trailing end of the long
              # line does end up getting sent anywhere, it will have
              # some indication that it is incomplete.
              bufend = "[...]"
            else
              # If line length is sane, we'll wait for the rest of the
              # line to appear in the next read_pipes() call.
              bufend = line
              break
            end
          end
          # rate_limit returns true or false as to whether to actually log
          # the line or not.  It also modifies "line" in place to replace
          # it with an error if a logging limit is tripped.
          if rate_limit j, line
            $stderr.print "#{job_uuid} ! " unless line.index(job_uuid)
            $stderr.puts line
            pub_msg = "#{LogTime.now} #{line.strip}\n"
            j[:stderr_buf_to_flush] << pub_msg
          end
        end

        # Leave the trailing incomplete line (if any) in streambuf for
        # next time.
        streambuf.replace bufend
      end
      # Flush buffered logs to the logs table, if appropriate. We have
      # to do this even if we didn't collect any new logs this time:
      # otherwise, buffered data older than seconds_between_events
      # won't get flushed until new data arrives.
      write_log j
    end
  end

  def reap_children
    return if 0 == @running.size
    pid_done = nil
    j_done = nil

    if false
      begin
        pid_done = waitpid(-1, Process::WNOHANG | Process::WUNTRACED)
        if pid_done
          j_done = @running.values.
            select { |j| j[:wait_thr].pid == pid_done }.
            first
        end
      rescue SystemCallError
        # I have @running processes but system reports I have no
        # children. This is likely to happen repeatedly if it happens at
        # all; I will log this no more than once per child process I
        # start.
        if 0 < @running.select { |uuid,j| j[:warned_waitpid_error].nil? }.size
          children = @running.values.collect { |j| j[:wait_thr].pid }.join ' '
          $stderr.puts "dispatch: IPC bug: waitpid() error (#{$!}), but I have children #{children}"
        end
        @running.each do |uuid,j| j[:warned_waitpid_error] = true end
      end
    else
      @running.each do |uuid, j|
        if j[:wait_thr].status == false
          pid_done = j[:wait_thr].pid
          j_done = j
        end
      end
    end

    return if !pid_done

    job_done = j_done[:job]

    # Ensure every last drop of stdout and stderr is consumed.
    read_pipes
    # Reset flush timestamp to make sure log gets written.
    j_done[:stderr_flushed_at] = Time.new(0)
    # Write any remaining logs.
    write_log j_done

    j_done[:buf].each do |stream, streambuf|
      if streambuf != ''
        $stderr.puts streambuf + "\n"
      end
    end

    # Wait the thread (returns a Process::Status)
    exit_status = j_done[:wait_thr].value.exitstatus
    exit_tempfail = exit_status == EXIT_TEMPFAIL

    $stderr.puts "dispatch: child #{pid_done} exit #{exit_status}"
    $stderr.puts "dispatch: job #{job_done.uuid} end"

    jobrecord = Job.find_by_uuid(job_done.uuid)

    if exit_status == EXIT_RETRY_UNLOCKED
      # The job failed because all of the nodes allocated to it
      # failed.  Only this crunch-dispatch process can retry the job:
      # it's already locked, and there's no way to put it back in the
      # Queued state.  Put it in our internal todo list unless the job
      # has failed this way excessively.
      @job_retry_counts[jobrecord.uuid] += 1
      exit_tempfail = @job_retry_counts[jobrecord.uuid] <= RETRY_UNLOCKED_LIMIT
      if exit_tempfail
        @todo_job_retries[jobrecord.uuid] = jobrecord
      else
        $stderr.puts("dispatch: job #{jobrecord.uuid} exceeded node failure retry limit -- giving up")
      end
    end

    if !exit_tempfail
      @job_retry_counts.delete(jobrecord.uuid)
      if jobrecord.state == "Running"
        # Apparently there was an unhandled error.  That could potentially
        # include "all allocated nodes failed" when we don't to retry
        # because the job has already been retried RETRY_UNLOCKED_LIMIT
        # times.  Fail the job.
        jobrecord.state = "Failed"
        if not jobrecord.save
          $stderr.puts "dispatch: jobrecord.save failed"
        end
      end
    else
      # If the job failed to run due to an infrastructure
      # issue with crunch-job or slurm, we want the job to stay in the
      # queue. If crunch-job exited after losing a race to another
      # crunch-job process, it exits 75 and we should leave the job
      # record alone so the winner of the race can do its thing.
      # If crunch-job exited after all of its allocated nodes failed,
      # it exits 93, and we want to retry it later (see the
      # EXIT_RETRY_UNLOCKED `if` block).
      #
      # There is still an unhandled race condition: If our crunch-job
      # process is about to lose a race with another crunch-job
      # process, but crashes before getting to its "exit 75" (for
      # example, "cannot fork" or "cannot reach API server") then we
      # will assume incorrectly that it's our process's fault
      # jobrecord.started_at is non-nil, and mark the job as failed
      # even though the winner of the race is probably still doing
      # fine.
    end

    # Invalidate the per-job auth token, unless the job is still queued and we
    # might want to try it again.
    if jobrecord.state != "Queued" and !@todo_job_retries.include?(jobrecord.uuid)
      j_done[:job_auth].update_attributes expires_at: Time.now
    end

    @running.delete job_done.uuid
  end

  def update_pipelines
    expire_tokens = @pipe_auth_tokens.dup
    @todo_pipelines.each do |p|
      pipe_auth = (@pipe_auth_tokens[p.uuid] ||= ApiClientAuthorization.
                   create(user: User.where('uuid=?', p.modified_by_user_uuid).first,
                          api_client_id: 0))
      puts `export ARVADOS_API_TOKEN=#{pipe_auth.api_token} && arv-run-pipeline-instance --run-pipeline-here --no-wait --instance #{p.uuid}`
      expire_tokens.delete p.uuid
    end

    expire_tokens.each do |k, v|
      v.update_attributes expires_at: Time.now
      @pipe_auth_tokens.delete k
    end
  end

  def run
    act_as_system_user
    $stderr.puts "dispatch: ready"
    while !$signal[:term] or @running.size > 0
      read_pipes
      if $signal[:term]
        @running.each do |uuid, j|
          if !j[:started] and j[:sent_int] < 2
            begin
              Process.kill 'INT', j[:wait_thr].pid
            rescue Errno::ESRCH
              # No such pid = race condition + desired result is
              # already achieved
            end
            j[:sent_int] += 1
          end
        end
      else
        refresh_todo unless did_recently(:refresh_todo, 1.0)
        update_node_status unless did_recently(:update_node_status, 1.0)
        unless @todo.empty? or did_recently(:start_jobs, 1.0) or $signal[:term]
          start_jobs
        end
        unless (@todo_pipelines.empty? and @pipe_auth_tokens.empty?) or did_recently(:update_pipelines, 5.0)
          update_pipelines
        end
      end
      reap_children
      select(@running.values.collect { |j| [j[:stdout], j[:stderr]] }.flatten,
             [], [], 1)
    end
    # If there are jobs we wanted to retry, we have to mark them as failed now.
    # Other dispatchers can't pick them up because we hold their lock.
    @todo_job_retries.each_key do |job_uuid|
      job = Job.find_by_uuid(job_uuid)
      if job.state == "Running"
        fail_job(job, "crunch-dispatch was stopped during job's tempfail retry loop")
      end
    end
  end

  protected

  def did_recently(thing, min_interval)
    if !@did_recently[thing] or @did_recently[thing] < Time.now - min_interval
      @did_recently[thing] = Time.now
      false
    else
      true
    end
  end

  # send message to log table. we want these records to be transient
  def write_log running_job
    return if running_job[:stderr_buf_to_flush] == ''

    # Send out to log event if buffer size exceeds the bytes per event or if
    # it has been at least crunch_log_seconds_between_events seconds since
    # the last flush.
    if running_job[:stderr_buf_to_flush].size > Rails.configuration.crunch_log_bytes_per_event or
        (Time.now - running_job[:stderr_flushed_at]) >= Rails.configuration.crunch_log_seconds_between_events
      begin
        log = Log.new(object_uuid: running_job[:job].uuid,
                      event_type: 'stderr',
                      owner_uuid: running_job[:job].owner_uuid,
                      properties: {"text" => running_job[:stderr_buf_to_flush]})
        log.save!
        running_job[:events_logged] += 1
      rescue => exception
        $stderr.puts "Failed to write logs"
        $stderr.puts exception.backtrace
      end
      running_job[:stderr_buf_to_flush] = ''
      running_job[:stderr_flushed_at] = Time.now
    end
  end
end

# This is how crunch-job child procs know where the "refresh" trigger file is
ENV["CRUNCH_REFRESH_TRIGGER"] = Rails.configuration.crunch_refresh_trigger

# If salloc can't allocate resources immediately, make it use our temporary
# failure exit code.  This ensures crunch-dispatch won't mark a job failed
# because of an issue with node allocation.  This often happens when
# another dispatcher wins the race to allocate nodes.
ENV["SLURM_EXIT_IMMEDIATE"] = Dispatcher::EXIT_TEMPFAIL.to_s

Dispatcher.new.run
