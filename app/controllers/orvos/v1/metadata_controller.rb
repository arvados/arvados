class Orvos::V1::MetadataController < ApplicationController
  def index
    if params[:tail_kind] and params[:tail]
      params[:where] = JSON.parse(params[:where]) if params[:where].is_a?(String)
      params[:where] ||= {}
      params[:where][:tail_kind] = params[:tail_kind]
      params[:where][:tail] = params[:tail]
    end
    super
  end
end
