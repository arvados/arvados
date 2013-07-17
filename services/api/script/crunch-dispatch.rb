#!/usr/bin/env ruby

include Process

$signal = {}
%w{TERM INT}.each do |sig|
  signame = sig
  Signal.trap(sig) do
    $stderr.puts "Received #{signame} signal"
    $signal[:term] = true
  end
end

ENV["RAILS_ENV"] = ARGV[0] || ENV["RAILS_ENV"] || "development"

require File.dirname(__FILE__) + '/../config/boot'
require File.dirname(__FILE__) + '/../config/environment'
require 'open3'

$redis ||= Redis.new

class Dispatcher
  include ApplicationHelper

  def sysuser
    return act_as_system_user
  end

  def refresh_todo
    @todo = Job.queue
  end

  def start_jobs
    if Server::Application.config.crunch_job_wrapper.to_s.match /^slurm/
      @idle_slurm_nodes = 0
      begin
        `sinfo`.
          split("\n").
          collect { |line| line.match /(\d+) +idle/ }.
          each do |re|
          @idle_slurm_nodes = re[1].to_i if re
        end
      rescue
      end
    end

    @todo.each do |job|

      min_nodes = 1
      begin
        if job.resource_limits['min_nodes']
          min_nodes = begin job.resource_limits['min_nodes'].to_i rescue 1 end
        end
      end
      next if @idle_slurm_nodes and @idle_slurm_nodes < min_nodes

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

      cmd_args << (ENV['CRUNCH_JOB_BIN'] || `which crunch-job`.strip)
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
      $stderr.puts "dispatch: job #{job.uuid} start"
      $stderr.puts "dispatch: child #{t.pid} start"
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
            $redis.publish job_uuid, "#{Time.now.ctime.to_s} #{line.strip}"
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
        start_jobs unless @todo.empty? or did_recently(:start_jobs, 1.0)
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

Dispatcher.new.run
