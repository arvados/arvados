class NodesController < ApplicationController
  skip_before_filter :authenticate_api_token
  def index
    @objects = model_class.order("created_at desc")
  end
end
