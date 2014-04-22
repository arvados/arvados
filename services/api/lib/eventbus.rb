require 'eventmachine'
require 'oj'
require 'faye/websocket'
require 'record_filters'

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

class FilterController
  include RecordFilters

  def initialize(f, o)
    @filters = f
    @objects = o
    apply_where_limit_order_params
  end

  def each &b
    @objects.each &b
  end

  def params
    {}
  end

end

class EventBus
  include CurrentApiClient

  def initialize
    @channel = EventMachine::Channel.new
    @mtx = Mutex.new
    @bgthread = false
  end

  def on_connect ws
    if not current_user
      ws.send '{"error":"Not logged in"}'
      ws.close
      return
    end

    ws.user = current_user
    ws.filters = []

    sub = @channel.subscribe do |msg|
      Log.where(id: msg.to_i).each do |l|
        ws.last_log_id = msg.to_i
        if rsc = ArvadosModel::resource_class_for_uuid(l.object_uuid)
          permitted = rsc.readable_by(ws.user).where(uuid: l.object_uuid)
          ws.filters.each do |filter|
            FilterController.new(filter, permitted).each do
              ws.send(l.as_api_response.to_json)
            end
          end
        end
      end
    end

    ws.on :message do |event|
      ws.filters = Filter.new oj.parse(event.data)
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
