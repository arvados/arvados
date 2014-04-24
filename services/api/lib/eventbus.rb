require 'eventmachine'
require 'oj'
require 'faye/websocket'
require 'record_filters'
require 'load_param'

module Faye
  class WebSocket
    attr_accessor :user
    attr_accessor :last_log_id
    attr_accessor :filters
  end
end

class Filter
  include LoadParam

  def initialize p
    @p = p
    load_filters_param
  end

  def params
    @p
  end

  def filters
    @filters
  end
end

class EventBus
  include CurrentApiClient
  include RecordFilters

  def initialize
    @channel = EventMachine::Channel.new
    @mtx = Mutex.new
    @bgthread = false
  end

  def on_connect ws
    if not current_user
      ws.send ({status: 401, message: "Valid API token required"}.to_json)
      ws.close
      return
    end

    ws.user = current_user
    ws.filters = []
    ws.last_log_id = nil

    sub = @channel.subscribe do |msg|
      begin
        # Must have at least one filter set up to receive events
        if ws.filters.length > 0

          # Start with log rows readable by user, sorted in ascending order
          logs = Log.readable_by(ws.user).order("id asc")

          if ws.last_log_id
            # Only get log rows that are new
            logs = logs.where("logs.id > ? and logs.id <= ?", ws.last_log_id, msg.to_i)
          else
            # No last log id, so only look at the most recently changed row
            logs = logs.where("logs.id = ?", msg.to_i)
          end

          # Record the most recent row
          ws.last_log_id = msg.to_i

          # Now process filters provided by client
          cond_out = []
          param_out = []
          ws.filters.each do |filter|
            ft = record_filters filter.filters
            cond_out += ft[:cond_out]
            param_out += ft[:param_out]
          end

          # Add filters to query
          if cond_out.any?
            logs = logs.where(cond_out.join(' OR '), *param_out)
          end

          # Finally execute query and send matching rows
          logs.each do |l|
            ws.send(l.as_api_response.to_json)
          end
        else
          # No filters set up, so just record the sequence number
          ws.last_log_id.nil = msg.to_i
        end
      rescue Exception => e
        puts "#{e}"
        ws.close
      end
    end

    ws.on :message do |event|
      p = Oj.load event.data
      if p["method"] == 'subscribe'
        if p["starting_log_id"]
          ws.last_log_id = p["starting_log_id"].to_i
        end
        ws.filters.push(Filter.new p)
        ws.send ({status: 200, message: 'subscribe ok'}.to_json)
      end
    end

    ws.on :close do |event|
      @channel.unsubscribe sub
      ws = nil
    end

    @mtx.synchronize do
      unless @bgthread
        @bgthread = true
        Thread.new do
          # from http://stackoverflow.com/questions/16405520/postgres-listen-notify-rails
          ActiveRecord::Base.connection_pool.with_connection do |connection|
            conn = connection.instance_variable_get(:@connection)
            begin
              conn.async_exec "LISTEN logs"
              while true
                conn.wait_for_notify do |channel, pid, payload|
                  @channel.push payload
                end
              end
            ensure
              # Don't want the connection to still be listening once we return
              # it to the pool - could result in weird behavior for the next
              # thread to check it out.
              conn.async_exec "UNLISTEN *"
            end
          end
        end
      end
    end
  end
end
