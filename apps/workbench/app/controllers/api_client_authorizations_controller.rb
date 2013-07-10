class ApiClientAuthorizationsController < ApplicationController
  def index
    @objects = model_class.all.to_ary.reject do |x|
      x.api_client_id == 0 or (x.expires_at and x.expires_at < Time.now) rescue false
    end
    super
  end
end
