# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'current_api_client'
require 'db_current_time'

module AuditLogs
  extend CurrentApiClient
  extend DbCurrentTime

  def self.delete_old(max_age:, max_batch:)
    act_as_system_user do
      if !File.owned?(Rails.root.join('tmp'))
        Rails.logger.warn("AuditLogs: not owner of #{Rails.root}/tmp, skipping")
        return
      end
      lockfile = Rails.root.join('tmp', 'audit_logs.lock')
      File.open(lockfile, File::RDWR|File::CREAT, 0600) do |f|
        return unless f.flock(File::LOCK_NB|File::LOCK_EX)

        sql = "select clock_timestamp() - interval '#{'%.9f' % max_age} seconds'"
        threshold = ActiveRecord::Base.connection.select_value(sql).to_time.utc
        Rails.logger.info "AuditLogs: deleting logs older than #{threshold}"

        did_total = 0
        loop do
          sql = Log.unscoped.
                select(:id).
                order(:created_at).
                where('event_type in (?)', ['create', 'update', 'destroy', 'delete']).
                where('created_at < ?', threshold).
                limit(max_batch).
                to_sql
          did = Log.unscoped.where("id in (#{sql})").delete_all
          did_total += did

          Rails.logger.info "AuditLogs: deleted batch of #{did}"
          break if did == 0
        end
        Rails.logger.info "AuditLogs: deleted total #{did_total}"
      end
    end
  end

  def self.tidy_in_background
    max_age = Rails.configuration.max_audit_log_age
    max_batch = Rails.configuration.max_audit_log_delete_batch
    return if max_age <= 0 || max_batch <= 0

    exp = (max_age/14).seconds
    need = false
    Rails.cache.fetch('AuditLogs', expires_in: exp) do
      need = true
    end
    return if !need

    Thread.new do
      Thread.current.abort_on_exception = false
      begin
        delete_old(max_age: max_age, max_batch: max_batch)
      rescue => e
        Rails.logger.error "#{e.class}: #{e}\n#{e.backtrace.join("\n\t")}"
      ensure
        ActiveRecord::Base.connection.close
      end
    end
  end
end
