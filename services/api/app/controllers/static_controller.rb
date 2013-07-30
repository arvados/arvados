class StaticController < ApplicationController

  skip_before_filter :uncamelcase_params_hash_keys
  skip_before_filter :find_object_by_uuid
  skip_before_filter :require_auth_scope_all, :only => :home

  def home
    render 'intro'
  end

end
