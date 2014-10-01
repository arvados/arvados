class StaticController < ApplicationController
  respond_to :json, :html

  skip_before_filter :find_object_by_uuid
  skip_before_filter :render_404_if_no_object
  skip_before_filter :require_auth_scope, :only => [ :home, :login_failure ]

  def home
    respond_to do |f|
      f.html do
        if Rails.configuration.workbench_address
          redirect_to Rails.configuration.workbench_address
        else
          render_not_found "Oops, this is an API endpoint. You probably want to point your browser to an Arvados Workbench site instead."
        end
      end
      f.json do
        render_not_found "Path not found."
      end
    end
  end

end
