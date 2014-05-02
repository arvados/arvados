class WebsocketController < ApplicationController
  skip_before_filter :find_objects_for_index

  def index
  end

  def model_class
    "Websocket"
  end
end
