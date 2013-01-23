class StaticController < ApplicationController

  skip_before_filter :uncamelcase_params_hash_keys
  skip_before_filter :find_object_by_uuid
  skip_before_filter :authenticate_api_token
  skip_before_filter :login_required, :only => :home

  def home
    render 'intro'
  end

end
