class Arvados::V1::LinksController < ApplicationController

  prepend_before_filter :load_kind_params, :only => :index

  def create
    resource_attrs.delete :head_kind
    resource_attrs.delete :tail_kind
    super
  end

  def load_kind_params
    if params[:tail_uuid]
      params[:where] = Oj.load(params[:where]) if params[:where].is_a?(String)
      @where ||= {}
      @where[:tail_uuid] = params[:tail_uuid]
    end

    if params[:where] and params[:where].is_a? Hash
      if params[:where][:head_kind]
        params[:filters] ||= []
        params[:filters] << ['head_uuid', 'is_a', params[:where][:head_kind]]
        params[:where].delete :head_kind
      end
      if params[:where][:tail_kind]
        params[:filters] ||= []
        params[:filters] << ['tail_uuid', 'is_a', params[:where][:tail_kind]]
        params[:where].delete :tail_kind
      end
    end

  end

end
