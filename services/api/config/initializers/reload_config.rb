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

      t_lastload = Rails.configuration.SourceTimestamp
      hash_lastload = Rails.configuration.SourceSHA256
      conffile = ENV['ARVADOS_CONFIG'] || "/etc/arvados/config.yml"
      Rails.logger.info("reload_config: polling for updated mtime on #{conffile} with threshold #{t_lastload}")
      while true
        sleep 1
        t = File.mtime(conffile)
        # If the file is newer than 5s, re-read it even if the
        # timestamp matches the previously loaded file. This enables
        # us to detect changes even if the filesystem's timestamp
        # precision cannot represent multiple updates per second.
        if t.to_f != t_lastload.to_f || Time.now.to_f - t.to_f < 5
          Open3.popen2("arvados-server", "config-dump", "-skip-legacy") do |stdin, stdout, status_thread|
            confs = YAML.load(stdout, deserialize_symbols: false)
            hash = confs["SourceSHA256"]
          rescue => e
            Rails.logger.info("reload_config: config file updated but could not be loaded: #{e}")
            t_lastload = t
            continue
          end
          if hash == hash_lastload
            # If we reloaded a new or updated file, but the content is
            # identical, keep polling instead of restarting.
            t_lastload = t
            continue
          end

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
