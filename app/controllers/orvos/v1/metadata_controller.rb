class Orvos::V1::MetadataController < ApplicationController
  def index
    if params[:target_kind] and params[:target_uuid]
      @objects = Metadatum.where('target_kind=? and target_uuid=?',
                                 params[:target_kind], params[:target_uuid])
    end
    super
  end

  def create
    @m = params[:metadatum]
    if params[:metadatum].class == String
      @m = uncamelcase_hash_keys(JSON.parse params[:metadatum])
    end
    @m = Metadatum.new @m
    respond_to do |format|
      if @m.save
        format.html { redirect_to @m, notice: 'Metadatum was successfully created.' }
        format.json { render json: @m, status: :created, location: @m }
      else
        format.html { render action: "new" }
        format.json { render json: @m.errors, status: :unprocessable_entity }
      end
    end
  end

end
