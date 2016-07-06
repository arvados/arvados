# If any threads raise an unhandled exception, make them all die.
# We trust a supervisor like runit to restart the server in this case.
Thread.abort_on_exception = true

require 'eventmachine'
require 'oj'
require 'faye/websocket'
require 'record_filters'
require 'load_param'
require 'set'
require 'thread'

# Patch in user, last_log_id and filters fields into the Faye::Websocket class.
module Faye
  class WebSocket
    attr_accessor :user
    attr_accessor :last_log_id
    attr_accessor :filters
    attr_accessor :sent_ids
    attr_accessor :queue
    attr_accessor :frame_mtx
  end
end

module WebSocket
  class Driver

    class Server
      alias_method :_write, :write

      def write(data)
        # Most of the sending activity will be from the thread set up in
        # on_connect.  However, there is also some automatic activity in the
        # form of ping/pong messages, so ensure that the write method used to
        # send one complete message to the underlying socket can only be
        # called by one thread at a time.
        self.frame_mtx.synchronize do
          _write(data)
        end
      end
    end
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
    @connection_count = 0
  end

  # Push out any pending events to the connection +ws+
  # +notify_id+  the id of the most recent row in the log table, may be nil
  #
  # This accepts a websocket and a notify_id (this is the row id from Postgres
  # LISTEN/NOTIFY, it may be nil if called from somewhere else)
  #
  # It queries the database for log rows that are either
  #  a) greater than ws.last_log_id, which is the last log id which was a candidate to be sent out
  #  b) if ws.last_log_id is nil, then it queries the row notify_id
  #
  # Regular Arvados permissions are applied using readable_by() and filters using record_filters().
  def push_events ws, notify_id
    begin
      # Must have at least one filter set up to receive events
      if ws.filters.length > 0
        # Start with log rows readable by user
        logs = Log.readable_by(ws.user)

        cond_id = nil
        cond_out = []
        param_out = []

        if not ws.last_log_id.nil?
          # We are catching up from some starting point.
          cond_id = "logs.id > ?"
          param_out << ws.last_log_id
        elsif not notify_id.nil?
          # Get next row being notified.
          cond_id = "logs.id = ?"
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

        # Execute query and actually send the matching log rows. Load
        # the full log records only when we're ready to send them,
        # though: otherwise, (1) postgres has to build the whole
        # result set and return it to us before we can send the first
        # event, and (2) we store lots of records in memory while
        # waiting to spool them out to the client. Both of these are
        # troublesome when log records are large (e.g., a collection
        # update contains both old and new manifest_text).
        #
        # Note: find_each implies order('id asc'), which is what we
        # want.
        logs.select(:id).find_each do |l|
          if not ws.sent_ids.include?(l.id)
            # only send if not a duplicate
            ws.send(Log.find(l.id).as_api_response.to_json)
          end
          if not ws.last_log_id.nil?
            # record ids only when sending "catchup" messages, not notifies
            ws.sent_ids << l.id
          end
        end
        ws.last_log_id = nil
      end
    rescue ArgumentError => e
      # There was some kind of user error.
      Rails.logger.warn "Error publishing event: #{$!}"
      ws.send ({status: 500, message: $!}.to_json)
      ws.close
    rescue => e
      Rails.logger.warn "Error publishing event: #{$!}"
      Rails.logger.warn "Backtrace:\n\t#{e.backtrace.join("\n\t")}"
      ws.send ({status: 500, message: $!}.to_json)
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
        p = (Oj.strict_load event.data).symbolize_keys
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
          # Reset sent_ids for consistency
          # (always re-deliver all matching messages following last_log_id)
          ws.sent_ids = Set.new
        end

        if ws.filters.length < Rails.configuration.websocket_max_filters
          # Add a filter.  This gets the :filters field which is the same
          # format as used for regular index queries.
          ws.filters << filter
          ws.send ({status: 200, message: 'subscribe ok', filter: p}.to_json)

          # Send any pending events
          push_events ws, nil
        else
          ws.send ({status: 403, message: "maximum of #{Rails.configuration.websocket_max_filters} filters allowed per connection"}.to_json)
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

  def overloaded?
    @mtx.synchronize do
      @connection_count >= Rails.configuration.websocket_max_connections
    end
  end

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
    ws.sent_ids = Set.new
    ws.queue = Queue.new
    ws.frame_mtx = Mutex.new

    @mtx.synchronize do
      @connection_count += 1
    end

    # Subscribe to internal postgres notifications through @channel and
    # forward them to the thread associated with the connection.
    sub = @channel.subscribe do |msg|
      if ws.queue.length > Rails.configuration.websocket_max_notify_backlog
        ws.send ({status: 500, message: 'Notify backlog too long'}.to_json)
        ws.close
        @channel.unsubscribe sub
        ws.queue.clear
      else
        ws.queue << [:notify, msg]
      end
    end

    # Set up callback for inbound message dispatch.
    ws.on :message do |event|
      ws.queue << [:message, event]
    end

    # Set up socket close callback
    ws.on :close do |event|
      @channel.unsubscribe sub
      ws.queue.clear
      ws.queue << [:close, nil]
    end

    # Spin off a new thread to handle sending events to the client.  We need a
    # separate thread per connection so that a slow client doesn't interfere
    # with other clients.
    #
    # We don't want the loop in the request thread because on a TERM signal,
    # Puma waits for outstanding requests to complete, and long-lived websocket
    # connections may not complete in a timely manner.
    Thread.new do
      # Loop and react to socket events.
      begin
        loop do
          eventType, msg = ws.queue.pop
          if eventType == :message
            handle_message ws, msg
          elsif eventType == :notify
            push_events ws, msg
          elsif eventType == :close
            break
          end
        end
      ensure
        @mtx.synchronize do
          @connection_count -= 1
        end
      end
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

  end
end
