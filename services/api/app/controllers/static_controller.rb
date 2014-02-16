class StaticController < ApplicationController
  respond_to :json, :html

  skip_before_filter :find_object_by_uuid
  skip_before_filter :render_404_if_no_object
  skip_before_filter :require_auth_scope_all, :only => [ :home, :login_failure ]

  def home
    if Rails.configuration.respond_to? :workbench_address
      redirect_to Rails.configuration.workbench_address
    else
      render json: {
        error: ('This is the API server; you probably want to be at the workbench for this installation. Unfortunately, config.workbench_address is not set so I can not redirect you there automatically')
      }
    end
  end

end
