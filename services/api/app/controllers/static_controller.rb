class StaticController < ApplicationController
  respond_to :json, :html

  skip_before_filter :find_object_by_uuid
  skip_before_filter :render_404_if_no_object
  skip_before_filter :require_auth_scope_all, :only => [ :home, :login_failure ]

  def home
    redirect_to Rails.configuration.workbench_address
  end

end
