class StaticController < ApplicationController

  skip_before_filter :find_object_by_uuid
  skip_before_filter :require_auth_scope_all, :only => [ :home, :login_failure ]

  def home
    redirect_to Rails.configuration.workbench_address
  end

end
