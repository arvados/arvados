class NodesController < ApplicationController
  def index
    @objects = model_class.order("created_at desc")
  end
end
