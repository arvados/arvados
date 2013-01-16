class Orvos::V1::MetadataController < ApplicationController
  def index
    if params[:target_kind] and params[:target_uuid]
      @objects = Metadatum.where('target_kind=? and target_uuid=?',
                                 params[:target_kind], params[:target_uuid])
    end
    super
  end
end
