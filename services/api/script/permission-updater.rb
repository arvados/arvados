#!/usr/bin/env ruby

ENV["RAILS_ENV"] = ARGV[0] || ENV["RAILS_ENV"] || "development"
require File.dirname(__FILE__) + '/../config/boot'
require File.dirname(__FILE__) + '/../config/environment'
include DbCurrentTime

def update_permissions
  timestamp = DbCurrentTime::db_current_time.to_i
  Rails.logger.info "Begin updating permission cache"
  User.all.each do |u|
    u.calculate_group_permissions
  end
  Rails.cache.write "last_updated_permissions", timestamp
  Rails.logger.info "Permission cache updated"
end

ActiveRecord::Base.connection_pool.with_connection do |connection|
  conn = connection.instance_variable_get(:@connection)
  begin
    conn.async_exec "LISTEN invalidate_permissions_cache"

    # Initial refresh of permissions graph
    update_permissions

    while true
      # wait_for_notify will block until there is a change
      # notification from Postgres about the permission cache,
      # and then rebuild the permission cache.
      conn.wait_for_notify do |channel, pid, payload|
        last_updated = Rails.cache.read("last_updated_permissions")
        Rails.logger.info "Got notify #{payload} last update #{last_updated}"
        if last_updated.nil? || last_updated.to_i <= payload.to_i
          update_permissions
        end
      end
    end
  ensure
    # Don't want the connection to still be listening once we return
    # it to the pool - could result in weird behavior for the next
    # thread to check it out.
    conn.async_exec "UNLISTEN *"
  end
end
