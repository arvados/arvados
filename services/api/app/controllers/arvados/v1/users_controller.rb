class Arvados::V1::UsersController < ApplicationController
  def current
    @object = current_user
    show
  end
  def system
    @object = system_user
    show
  end

  class ChannelStreamer
    Q_UPDATE_INTERVAL = 12
    def initialize(opts={})
      @opts = opts
    end
    def each
      return unless @opts[:channel]
      @redis = Redis.new(:timeout => 0)
      @redis.subscribe(@opts[:channel]) do |event|
        event.message do |channel, msg|
          yield msg + "\n"
        end
      end
    end
  end
      
  def event_stream
    channel = current_user.andand.uuid
    if current_user.andand.is_admin
      channel = params[:uuid] || channel
    end
    if client_accepts_plain_text_stream
      self.response.headers['Last-Modified'] = Time.now.ctime.to_s
      self.response_body = ChannelStreamer.new(channel: channel)
    else
      render json: {
        href: url_for(uuid: channel),
        comment: ('To retrieve the event stream as plain text, ' +
                  'use a request header like "Accept: text/plain"')
      }
    end
  end
end
