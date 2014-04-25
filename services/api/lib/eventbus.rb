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

  attr_accessor :filters

  def initialize p, fid
    @params = p
    @filter_id = fid
    load_filters_param
  end

  def params
    @params
  end

  def filter_id
    @filter_id
  end
end

class EventBus
  include CurrentApiClient
  include RecordFilters

  # used in RecordFilters
  def model_class
    Log
  end

  # used in RecordFilters
  def table_name
    model_class.table_name
  end

  def initialize
    @channel = EventMachine::Channel.new
    @mtx = Mutex.new
    @bgthread = false
    @filter_id_counter = 0
  end

  def alloc_filter_id
    (@filter_id_counter += 1)
  end

  def push_events ws, msg = nil
      begin
        # Must have at least one filter set up to receive events
        if ws.filters.length > 0
          # Start with log rows readable by user, sorted in ascending order
          logs = Log.readable_by(ws.user).order("id asc")

          if ws.last_log_id
            # Only interested in log rows that are new
            logs = logs.where("logs.id > ?", ws.last_log_id)
          elsif msg
            # No last log id, so only look at the most recently changed row
            logs = logs.where("logs.id = ?", msg.to_i)
          else
            return
          end

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
            ws.last_log_id = l.id
          end
        elsif msg
          # No filters set up, so just record the sequence number
          ws.last_log_id = msg.to_i
        end
      rescue Exception => e
        puts "Error publishing event: #{$!}"
        puts "Backtrace:\n\t#{e.backtrace.join("\n\t")}"
        ws.send ({status: 500, message: 'error'}.to_json)
        ws.close
      end
  end

  MAX_FILTERS = 16

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
      push_events ws, msg
    end

    ws.on :message do |event|
      begin
        p = (Oj.load event.data).symbolize_keys
        if p[:method] == 'subscribe'
          if p[:last_log_id]
            ws.last_log_id = p[:last_log_id].to_i
          end

          if ws.filters.length < MAX_FILTERS
            filter_id = alloc_filter_id
            ws.filters.push Filter.new(p, filter_id)
            ws.send ({status: 200, message: 'subscribe ok', filter_id: filter_id}.to_json)
            push_events ws
          else
            ws.send ({status: 403, message: "maximum of #{MAX_FILTERS} filters allowed per connection"}.to_json)
          end
        elsif p[:method] == 'unsubscribe'
          if filter_id = p[:filter_id]
            filter_id = filter_id.to_i
            len = ws.filters.length
            ws.filters = ws.filters.select { |f| f.filter_id != filter_id }
            if ws.filters.length < len
              ws.send ({status: 200, message: 'unsubscribe ok', filter_id: filter_id}.to_json)
            else
              ws.send ({status: 404, message: 'filter_id not found', filter_id: filter_id}.to_json)
            end
          else
            ws.send ({status: 400, message: 'must provide filter_id'}.to_json)
          end
        else
          ws.send ({status: 400, message: "missing or unrecognized method"}.to_json)
        end
      rescue Oj::Error => e
        ws.send ({status: 400, message: "malformed request"}.to_json)
      rescue Exception => e
        puts "Error handling message: #{$!}"
        puts "Backtrace:\n\t#{e.backtrace.join("\n\t")}"
        ws.send ({status: 500, message: 'error'}.to_json)
        ws.close
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
