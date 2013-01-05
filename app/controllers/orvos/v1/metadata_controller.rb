class Orvos::V1::MetadataController < ApplicationController
  def index
    @metadata = Collection.all
    @metadatumlist = {
      :kind  => "orvos#metadatumList",
      :etag => "",
      :self_link => "",
      :next_page_token => "",
      :next_link => "",
      :items => @metadata.map { |x| x }
    }
    respond_to do |format|
      format.json { render json: @metadatumlist }
    end
  end

  def show
    @m = Metadatum.find(params[:id])

    respond_to do |format|
      format.json { render json: @m }
    end
  end

  def create
    if params[:metadatum].class == String
      @m = Metadatum.new(JSON.parse(params[:metadatum]))
    else
      @m = Metadatum.new(params[:metadatum])
    end
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
