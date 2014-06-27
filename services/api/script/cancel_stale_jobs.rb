#!/usr/bin/env ruby


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

class CancelJobs
  include ApplicationHelper

  def cancel_stale_jobs
    act_as_system_user do
      Job.running.each do |jobrecord|
        f = Log.where("object_uuid=?", jobrecord.uuid).limit(1).order("created_at desc").first
        if f
          age = (Time.now - f.created_at)
          if age > 300
            $stderr.puts "dispatch: failing orphan job #{jobrecord.uuid}, last log is #{age} seconds old"
            # job is marked running, but not known to crunch-dispatcher, and
            # hasn't produced any log entries for 5 minutes, so mark it as failed.
            jobrecord.running = false
            jobrecord.cancelled_at ||= Time.now
            jobrecord.finished_at ||= Time.now
            if jobrecord.success.nil?
              jobrecord.success = false
            end
            jobrecord.save!
          end
        end
      end
    end
  end
end

CancelJobs.new.cancel_stale_jobs
