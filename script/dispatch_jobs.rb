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

ENV["RAILS_ENV"] = ARGV[0] || "development"

require File.dirname(__FILE__) + '/../config/boot'
require File.dirname(__FILE__) + '/../config/environment'
require 'open3'

class Dispatcher

  def sysuser
    return @sysuser if @sysuser
    Thread.current[:user] = User.new(is_admin: true)
    sysuser_id = [Server::Application.config.uuid_prefix,
                  User.uuid_prefix,
                  '000000000000000'].join('-')
    @sysuser = User.where('uuid=?', sysuser_id).first
    if !@sysuser
      @sysuser = User.new(uuid: sysuser_id,
                          is_admin: true,
                          email: 'root',
                          first_name: 'root',
                          last_name: '')
      @sysuser.save!
      @sysuser.reload
    end
    Thread.current[:user] = @sysuser

    auth = ApiClientAuthorization.new(api_client_id: 0,
                                      user_id: @sysuser.id)
    auth.save!
    auth_token = auth.api_token
    $stderr.puts "dispatch: sysuser.uuid = #{@sysuser.uuid}"
    $stderr.puts "dispatch: api_client_authorization.api_token = #{auth_token}"
    @sysuser
  end

  def refresh_todo
    @todo = Job.
      where('started_at is ? and is_locked_by is ? and cancelled_at is ?',
            nil, nil, nil).
      order('priority desc, created_at')
  end

  def start_jobs
    if Server::Application.config.whjobmanager_wrapper.to_s.match /^slurm/
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

      min_nodes = begin job.resource_limits['min_nodes'].to_i rescue 1 end
      next if @idle_slurm_nodes and @idle_slurm_nodes < min_nodes

      next if @running[job.uuid]
      next if !take(job)

      cmd_args = nil
      case Server::Application.config.whjobmanager_wrapper
      when :none
        cmd_args = []
      when :slurm_immediate
        cmd_args = ["salloc",
                    "--immediate",
                    "--exclusive",
                    "--no-kill",
                    "--job-name=#{job.uuid}",
                    "--nodes=1"]
      else
        raise "Unknown whjobmanager_wrapper: #{Server::Application.config.whjobmanager_wrapper}"
      end

      cmd_args << 'whjobmanager'
      cmd_args << "id=#{job.uuid}"
      cmd_args << "mrfunction=#{job.command}"
      job.command_parameters.each do |k,v|
        k = k.to_s
        if k == 'input'
          k = 'inputkey'
        else
          k = k.upcase
        end
        cmd_args << "#{k}=#{v}"
      end
      cmd_args << "revision=#{job.command_version}"

      begin
        cmd_args << "stepspernode=#{job.resource_limits['max_tasks_per_node'].to_i}"
      rescue
        # OK if limit is not specified. OK to ignore if not integer.
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
        sent_int: 0
      }
      i.close
    end
  end

  def take(job)
    lock_ok = false
    ActiveRecord::Base.transaction do
      job.reload
      if job.is_locked_by.nil? and
          job.update_attributes(is_locked_by: sysuser.uuid)
        lock_ok = true
      end
    end
    lock_ok
  end

  def untake(job)
    job.reload
    job.update_attributes is_locked_by: nil
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
          lines = j[:stderr_buf].lines "\n"
          if j[:stderr_buf][-1] == "\n"
            j[:stderr_buf] = ''
          else
            j[:stderr_buf] = lines.pop
          end
          lines.each do |line|
            $stderr.print "#{job_uuid} ! " unless line.index(job_uuid)
            $stderr.puts line
            line.chomp!
            if (re = line.match(/#{job_uuid} (\d+) (\S*) (.*)/))
              ignorethis, whjmpid, taskid, message = re.to_a
              if taskid == '' and message == 'start'
                $stderr.puts "dispatch: noticed #{job_uuid} started"
                j[:started] = true
                ActiveRecord::Base.transaction do
                  j[:job].reload
                  j[:job].update_attributes running: true
                end
              elsif taskid == '' and (re = message.match /^outputkey (\S+)$/)
                $stderr.puts "dispatch: noticed #{job_uuid} output #{re[1]}"
                j[:output] = re[1]
              elsif taskid == '' and (re = message.match /^meta key is (\S+)$/)
                $stderr.puts "dispatch: noticed #{job_uuid} log #{re[1]}"
                j[:log] = re[1]
                ActiveRecord::Base.transaction do
                  j[:job].reload
                  j[:job].update_attributes log: j[:log]
                end
              elsif taskid.match(/^\d+/) and (re = message.match /^failure /)
                $stderr.puts "dispatch: noticed #{job_uuid} task fail"
                ActiveRecord::Base.transaction do
                  j[:job].reload
                  j[:job].tasks_summary ||= {}
                  j[:job].tasks_summary[:failed] ||= 0
                  j[:job].tasks_summary[:failed] += 1
                  j[:job].save
                end
              elsif (re = message.match(/^status: (\d+) done, (\d+) running, (\d+) todo/))
                $stderr.puts "dispatch: noticed #{job_uuid} #{message}"
                ActiveRecord::Base.transaction do
                  j[:job].reload
                  j[:job].tasks_summary ||= {}
                  j[:job].tasks_summary[:done] = re[1].to_i
                  j[:job].tasks_summary[:running] = re[2].to_i
                  j[:job].tasks_summary[:todo] = re[3].to_i
                  j[:job].save
                end
                if re[2].to_i == 0 and re[3].to_i == 0
                  $stderr.puts "dispatch: noticed #{job_uuid} succeeded"
                  j[:success] = true
                end
              end
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

    # Ensure every last drop of stdout and stderr is consumed
    read_pipes
    if j_done[:stderr_buf] and j_done[:stderr_buf] != ''
      $stderr.puts j_done[:stderr_buf] + "\n"
    end

    j_done[:wait_thr].value          # wait the thread

    if !j_done[:started]
      # If the job never really started (due to a scheduling
      # failure), just put it back in the queue
      untake(job_done)
      $stderr.puts "dispatch: job #{job_done.uuid} requeued"
    else
      # Otherwise, mark the job as finished
      ActiveRecord::Base.transaction do
        job_done.reload
        job_done.log = j_done[:log]
        job_done.output = j_done[:output]
        job_done.success = j_done[:success]
        job_done.assert_finished
      end
    end
    @running.delete job_done.uuid
  end

  def run
    sysuser
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
