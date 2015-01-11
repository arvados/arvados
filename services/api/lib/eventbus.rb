# If any threads raise an unhandled exception, make them all die.
# We trust a supervisor like runit to restart the server in this case.
Thread.abort_on_exception = true

require 'eventmachine'
require 'oj'
require 'faye/websocket'
require 'record_filters'
require 'load_param'

# Patch in user, last_log_id and filters fields into the Faye::Websocket class.
module Faye
  class WebSocket
    attr_accessor :user
    attr_accessor :last_log_id
    attr_accessor :filters
  end
end

# Store the filters supplied by the user that will be applied to the logs table
# to determine which events to return to the listener.
class Filter
  include LoadParam

  attr_accessor :filters

  def initialize p
    @params = p
    load_filters_param
  end

  def params
    @params
  end
end

# Manages websocket connections, accepts subscription messages and publishes
# log table events.
class EventBus
  include CurrentApiClient
  include RecordFilters

  # used in RecordFilters
  def model_class
    Log
  end

  # Initialize EventBus.  Takes no parameters.
  def initialize
    @channel = EventMachine::Channel.new
    @mtx = Mutex.new
    @bgthread = false
  end

  # Push out any pending events to the connection +ws+
  # +notify_id+  the id of the most recent row in the log table, may be nil
  #
  # This accepts a websocket and a notify_id (this is the row id from Postgres
  # LISTEN/NOTIFY, it may be nil if called from somewhere else)
  #
  # It queries the database for log rows that are either
  #  a) greater than ws.last_log_id, which is the last log id which was a candidate to be sent out
  #  b) if ws.last_log_id is nil, then it queries rows starting with notify_id
  #
  # Regular Arvados permissions are applied using readable_by() and filters using record_filters()
  # To avoid clogging up the database, queries are limited to batches of 100.  It will schedule a new
  # push_events call if there are more log rows to send.
  def push_events ws, notify_id
    begin
      if !notify_id.nil? and !ws.last_log_id.nil? and notify_id <= ws.last_log_id
        # This notify is for a row we've handled already.
        return
      end

      # Must have at least one filter set up to receive events
      if ws.filters.length > 0
        # Start with log rows readable by user, sorted in ascending order
        logs = Log.readable_by(ws.user).order("id asc")

        cond_id = nil
        cond_out = []
        param_out = []

        if !ws.last_log_id.nil?
          # Client is only interested in log rows that are newer than the
          # last log row seen by the client.
          cond_id = "logs.id > ?"
          param_out << ws.last_log_id
        elsif !notify_id.nil?
          # No last log id, so look at rows starting with notify id
          cond_id = "logs.id >= ?"
          param_out << notify_id
        else
          # No log id to start from, nothing to do, return
          return
        end

        # Now build filters provided by client
        ws.filters.each do |filter|
          ft = record_filters filter.filters, Log
          if ft[:cond_out].any?
            # Join the clauses within a single subscription filter with AND
            # so it is consistent with regular queries
            cond_out << "(#{ft[:cond_out].join ') AND ('})"
            param_out += ft[:param_out]
          end
        end

        # Add filters to query
        if cond_out.any?
          # Join subscriptions with OR
          logs = logs.where(cond_id + " AND ((#{cond_out.join ') OR ('}))", *param_out)
        else
          logs = logs.where(cond_id, *param_out)
        end

        # Execute query and actually send the matching log rows
        count = 0
        limit = 20

        logs.limit(limit).each do |l|
          ws.send(l.as_api_response.to_json)
          ws.last_log_id = l.id
          count += 1
        end

        if count == limit
          # Number of rows returned was capped by limit(), we need to schedule
          # another query to get more logs (will start from last_log_id
          # reported by current query)
          EventMachine::schedule do
            push_events ws, nil
          end
        elsif !notify_id.nil? and (ws.last_log_id.nil? or notify_id > ws.last_log_id)
          # Number of rows returned was less than cap, but the notify id is
          # higher than the last id visible to the client, so update last_log_id
          ws.last_log_id = notify_id
        end
      elsif !notify_id.nil?
        # No filters set up, so just record the sequence number
        ws.last_log_id = notify_id
      end
    rescue => e
      Rails.logger.warn "Error publishing event: #{$!}"
      Rails.logger.warn "Backtrace:\n\t#{e.backtrace.join("\n\t")}"
      ws.send ({status: 500, message: 'error'}.to_json)
      ws.close
      # These exceptions typically indicate serious server trouble:
      # out of memory issues, database connection problems, etc.  Go ahead and
      # crash; we expect that a supervisor service like runit will restart us.
      raise
    end
  end

  # Handle inbound subscribe or unsubscribe message.
  def handle_message ws, event
    begin
      begin
        # Parse event data as JSON
        p = (Oj.load event.data).symbolize_keys
        filter = Filter.new(p)
      rescue Oj::Error => e
        ws.send ({status: 400, message: "malformed request"}.to_json)
        return
      end

      if p[:method] == 'subscribe'
        # Handle subscribe event

        if p[:last_log_id]
          # Set or reset the last_log_id.  The event bus only reports events
          # for rows that come after last_log_id.
          ws.last_log_id = p[:last_log_id].to_i
        end

        if ws.filters.length < MAX_FILTERS
          # Add a filter.  This gets the :filters field which is the same
          # format as used for regular index queries.
          ws.filters << filter
          ws.send ({status: 200, message: 'subscribe ok', filter: p}.to_json)

          # Send any pending events
          push_events ws, nil
        else
          ws.send ({status: 403, message: "maximum of #{MAX_FILTERS} filters allowed per connection"}.to_json)
        end

      elsif p[:method] == 'unsubscribe'
        # Handle unsubscribe event

        len = ws.filters.length
        ws.filters.select! { |f| not ((f.filters == p[:filters]) or (f.filters.empty? and p[:filters].nil?)) }
        if ws.filters.length < len
          ws.send ({status: 200, message: 'unsubscribe ok'}.to_json)
        else
          ws.send ({status: 404, message: 'filter not found'}.to_json)
        end

      else
        ws.send ({status: 400, message: "missing or unrecognized method"}.to_json)
      end
    rescue => e
      Rails.logger.warn "Error handling message: #{$!}"
      Rails.logger.warn "Backtrace:\n\t#{e.backtrace.join("\n\t")}"
      ws.send ({status: 500, message: 'error'}.to_json)
      ws.close
    end
  end

  # Constant maximum number of filters, to avoid silly huge database queries.
  MAX_FILTERS = 16

  # Called by RackSocket when a new websocket connection has been established.
  def on_connect ws

    # Disconnect if no valid API token.
    # current_user is included from CurrentApiClient
    if not current_user
      ws.send ({status: 401, message: "Valid API token required"}.to_json)
      ws.close
      return
    end

    # Initialize our custom fields on the websocket connection object.
    ws.user = current_user
    ws.filters = []
    ws.last_log_id = nil

    # Subscribe to internal postgres notifications through @channel.  This will
    # call push_events when a notification comes through.
    sub = @channel.subscribe do |msg|
      push_events ws, msg
    end

    # Set up callback for inbound message dispatch.
    ws.on :message do |event|
      handle_message ws, event
    end

    # Set up socket close callback
    ws.on :close do |event|
      @channel.unsubscribe sub
      ws = nil
    end

    # Start up thread to monitor the Postgres database, if none exists already.
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
                # wait_for_notify will block until there is a change
                # notification from Postgres about the logs table, then push
                # the notification into the EventMachine channel.  Each
                # websocket connection subscribes to the other end of the
                # channel and calls #push_events to actually dispatch the
                # events to the client.
                conn.wait_for_notify do |channel, pid, payload|
                  @channel.push payload.to_i
                end
              end
            ensure
              # Don't want the connection to still be listening once we return
              # it to the pool - could result in weird behavior for the next
              # thread to check it out.
              conn.async_exec "UNLISTEN *"
            end
          end
          @bgthread = false
        end
      end
    end

    # Since EventMachine is an asynchronous event based dispatcher, #on_connect
    # does not block but instead returns immediately after having set up the
    # websocket and notification channel callbacks.
  end
end
