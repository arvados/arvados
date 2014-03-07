#!/usr/bin/env ruby

include Process

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

$redis ||= Redis.new
LOG_BUFFER_SIZE = 2**20

class Dispatcher
  include ApplicationHelper

  def sysuser
    return act_as_system_user
  end

  def refresh_todo
    @todo = Job.queue
  end

  def sinfo
    @@slurm_version ||= Gem::Version.new(`sinfo --version`.match(/\b[\d\.]+\b/)[0])
    if Gem::Version.new('2.3') <= @@slurm_version
      `sinfo --noheader -o '%n:%t'`.strip
    else
      # Expand rows with hostname ranges (like "foo[1-3,5,9-12]:idle")
      # into multiple rows with one hostname each.
      `sinfo --noheader -o '%N:%t'`.split("\n").collect do |line|
        tokens = line.split ":"
        if (re = tokens[0].match /^(.*?)\[([-,\d]+)\]$/)
          re[2].split(",").collect do |range|
            range = range.split("-").collect(&:to_i)
            (range[0]..range[-1]).collect do |n|
              [re[1] + n.to_s, tokens[1..-1]].join ":"
            end
          end
        else
          tokens.join ":"
        end
      end.flatten.join "\n"
    end
  end

  def update_node_status
    if Server::Application.config.crunch_job_wrapper.to_s.match /^slurm/
      @nodes_in_state = {idle: 0, alloc: 0, down: 0}
      @node_state ||= {}
      node_seen = {}
      begin
        sinfo.split("\n").
          each do |line|
          re = line.match /(\S+?):+(idle|alloc|down)/
          next if !re

          # sinfo tells us about a node N times if it is shared by N partitions
          next if node_seen[re[1]]
          node_seen[re[1]] = true

          # count nodes in each state
          @nodes_in_state[re[2].to_sym] += 1

          # update our database (and cache) when a node's state changes
          if @node_state[re[1]] != re[2]
            @node_state[re[1]] = re[2]
            node = Node.where('hostname=?', re[1]).first
            if node
              $stderr.puts "dispatch: update #{re[1]} state to #{re[2]}"
              node.info[:slurm_state] = re[2]
              node.save
            elsif re[2] != 'down'
              $stderr.puts "dispatch: sinfo reports '#{re[1]}' is not down, but no node has that name"
            end
          end
        end
      rescue
      end
    end
  end

  def start_jobs
    @todo.each do |job|

      min_nodes = 1
      begin
        if job.runtime_constraints['min_nodes']
          min_nodes = begin job.runtime_constraints['min_nodes'].to_i rescue 1 end
        end
      end

      begin
        next if @nodes_in_state[:idle] < min_nodes
      rescue
      end

      next if @running[job.uuid]
      next if !take(job)

      cmd_args = nil
      case Server::Application.config.crunch_job_wrapper
      when :none
        cmd_args = []
      when :slurm_immediate
        cmd_args = ["salloc",
                    "--chdir=/",
                    "--immediate",
                    "--exclusive",
                    "--no-kill",
                    "--job-name=#{job.uuid}",
                    "--nodes=#{min_nodes}"]
      else
        raise "Unknown crunch_job_wrapper: #{Server::Application.config.crunch_job_wrapper}"
      end

      if Server::Application.config.crunch_job_user
        cmd_args.unshift("sudo", "-E", "-u",
                         Server::Application.config.crunch_job_user,
                         "PERLLIB=#{ENV['PERLLIB']}")
      end

      job_auth = ApiClientAuthorization.
        new(user: User.where('uuid=?', job.modified_by_user_uuid).first,
            api_client_id: 0)
      job_auth.save

      crunch_job_bin = (ENV['CRUNCH_JOB_BIN'] || `which arv-crunch-job`.strip)
      if crunch_job_bin == ''
        raise "No CRUNCH_JOB_BIN env var, and crunch-job not in path."
      end

      cmd_args << crunch_job_bin
      cmd_args << '--job-api-token'
      cmd_args << job_auth.api_token
      cmd_args << '--job'
      cmd_args << job.uuid

      commit = Commit.where(sha1: job.script_version).first
      if commit
        cmd_args << '--git-dir'
        if File.exists?(File.
                        join(Rails.configuration.git_repositories_dir,
                             commit.repository_name + '.git'))
          cmd_args << File.
            join(Rails.configuration.git_repositories_dir,
                 commit.repository_name + '.git')
        else
          cmd_args << File.
            join(Rails.configuration.git_repositories_dir,
                 commit.repository_name, '.git')
        end
      end

      $stderr.puts "dispatch: #{cmd_args.join ' '}"

      begin
        i, o, e, t = Open3.popen3(*cmd_args)
      rescue
        $stderr.puts "dispatch: popen3: #{$!}"
        sleep 1
        untake(job)
        next
      end

      $stderr.puts "dispatch: job #{job.uuid}"
      start_banner = "dispatch: child #{t.pid} start #{Time.now.ctime.to_s}"
      $stderr.puts start_banner
      $redis.set job.uuid, start_banner + "\n"
      $redis.publish job.uuid, start_banner
      $redis.publish job.owner_uuid, start_banner

      @running[job.uuid] = {
        stdin: i,
        stdout: o,
        stderr: e,
        wait_thr: t,
        job: job,
        stderr_buf: '',
        started: false,
        sent_int: 0,
        job_auth: job_auth
      }
      i.close
    end
  end

  def take(job)
    # no-op -- let crunch-job take care of locking.
    true
  end

  def untake(job)
    # no-op -- let crunch-job take care of locking.
    true
  end

  def read_pipes
    @running.each do |job_uuid, j|
      job = j[:job]

      # Throw away child stdout
      begin
        j[:stdout].read_nonblock(2**20)
      rescue Errno::EAGAIN, EOFError
      end

      # Read whatever is available from child stderr
      stderr_buf = false
      begin
        stderr_buf = j[:stderr].read_nonblock(2**20)
      rescue Errno::EAGAIN, EOFError
      end

      if stderr_buf
        j[:stderr_buf] << stderr_buf
        if j[:stderr_buf].index "\n"
          lines = j[:stderr_buf].lines("\n").to_a
          if j[:stderr_buf][-1] == "\n"
            j[:stderr_buf] = ''
          else
            j[:stderr_buf] = lines.pop
          end
          lines.each do |line|
            $stderr.print "#{job_uuid} ! " unless line.index(job_uuid)
            $stderr.puts line
            pub_msg = "#{Time.now.ctime.to_s} #{line.strip}"
            $redis.publish job.owner_uuid, pub_msg
            $redis.publish job_uuid, pub_msg
            $redis.append job_uuid, pub_msg + "\n"
            if LOG_BUFFER_SIZE < $redis.strlen(job_uuid)
              $redis.set(job_uuid,
                         $redis
                           .getrange(job_uuid, (LOG_BUFFER_SIZE >> 1), -1)
                           .sub(/^.*?\n/, ''))
            end
          end
        end
      end
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
    $stderr.puts "dispatch: child #{pid_done} exit"
    $stderr.puts "dispatch: job #{job_done.uuid} end"
    $redis.publish job_done.uuid, "end"

    # Ensure every last drop of stdout and stderr is consumed
    read_pipes
    if j_done[:stderr_buf] and j_done[:stderr_buf] != ''
      $stderr.puts j_done[:stderr_buf] + "\n"
    end

    # Wait the thread
    j_done[:wait_thr].value

    # Invalidate the per-job auth token
    j_done[:job_auth].update_attributes expires_at: Time.now

    @running.delete job_done.uuid
  end

  def run
    act_as_system_user
    @running ||= {}
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
        update_node_status
        unless @todo.empty? or did_recently(:start_jobs, 1.0) or $signal[:term]
          start_jobs
        end
      end
      reap_children
      select(@running.values.collect { |j| [j[:stdout], j[:stderr]] }.flatten,
             [], [], 1)
    end
  end

  protected

  def did_recently(thing, min_interval)
    @did_recently ||= {}
    if !@did_recently[thing] or @did_recently[thing] < Time.now - min_interval
      @did_recently[thing] = Time.now
      false
    else
      true
    end
  end
end

# This is how crunch-job child procs know where the "refresh" trigger file is
ENV["CRUNCH_REFRESH_TRIGGER"] = Rails.configuration.crunch_refresh_trigger

Dispatcher.new.run
