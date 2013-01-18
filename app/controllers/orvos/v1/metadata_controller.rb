class Orvos::V1::MetadataController < ApplicationController
  def index
    if params[:tail_kind] and params[:tail]
      @objects = Metadatum.where('tail_kind=? and tail=?',
                                 params[:tail_kind], params[:tail])
    end
    super
  end
end
