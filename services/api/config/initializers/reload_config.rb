# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

if !File.owned?(Rails.root.join('tmp'))
  Rails.logger.debug("reload_config: not owner of #{Rails.root}/tmp, skipping")
elsif ENV["ARVADOS_CONFIG"] == "none"
  Rails.logger.debug("reload_config: no config in use, skipping")
else
  Thread.new do
    lockfile = Rails.root.join('tmp', 'reload_config.lock')
    File.open(lockfile, File::WRONLY|File::CREAT, 0600) do |f|
      # Note we don't use LOCK_NB here. If we did, each time passenger
      # kills the lock-holder process, we would be left with nobody
      # checking for updates until passenger starts a new worker,
      # which could be a long time.
      Rails.logger.debug("reload_config: waiting for lock on #{lockfile}")
      f.flock(File::LOCK_EX)
      conffile = ENV['ARVADOS_CONFIG'] || "/etc/arvados/config.yml"
      Rails.logger.info("reload_config: polling for updated mtime on #{conffile} with threshold #{Rails.configuration.SourceTimestamp}")
      while true
        sleep 1
        t = File.mtime(conffile)
        if t.to_f > Rails.configuration.SourceTimestamp.to_f
          restartfile = Rails.root.join('tmp', 'restart.txt')
          touchtime = Time.now
          Rails.logger.info("reload_config: mtime on #{conffile} changed to #{t}, touching #{restartfile} to #{touchtime}")
          File.utime(touchtime, touchtime, restartfile)
          # Even if passenger doesn't notice that we hit restart.txt
          # and kill our process, there's no point waiting around to
          # hit it again.
          break
        end
      end
    end
  end
end
