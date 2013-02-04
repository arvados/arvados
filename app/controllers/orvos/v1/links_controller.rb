class Orvos::V1::LinksController < ApplicationController
  def index
    if params[:tail_uuid]
      params[:where] = MultiJson.load(params[:where]) if params[:where].is_a?(String)
      params[:where] ||= {}
      params[:where][:tail_uuid] = params[:tail_uuid]
    end
    super
  end
end
