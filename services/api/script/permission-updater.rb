#!/usr/bin/env ruby

dispatch_argv = []
ARGV.reject! do |arg|
  dispatch_argv.push(arg) if /^--/ =~ arg
end

ENV["RAILS_ENV"] = ARGV[0] || ENV["RAILS_ENV"] || "development"
require File.dirname(__FILE__) + '/../config/boot'
require File.dirname(__FILE__) + '/../config/environment'

User.all.each do |u|
  u.calculate_group_permissions
end

ActiveRecord::Base.connection_pool.with_connection do |connection|
  conn = connection.instance_variable_get(:@connection)
  begin
    conn.async_exec "LISTEN invalidate_permissions_cache"
    while true
      # wait_for_notify will block until there is a change
      # notification from Postgres about the logs table, then push
      # the notification into the EventMachine channel.  Each
      # websocket connection subscribes to the other end of the
      # channel and calls #push_events to actually dispatch the
      # events to the client.
      conn.wait_for_notify do |channel, pid, payload|
        Rails.logger.info "Begin updating permission cache"
        User.all.each do |u|
          u.calculate_group_permissions
        end
        Rails.logger.info "Permission cache updated"
      end
    end
  ensure
    # Don't want the connection to still be listening once we return
    # it to the pool - could result in weird behavior for the next
    # thread to check it out.
    conn.async_exec "UNLISTEN *"
  end
end
