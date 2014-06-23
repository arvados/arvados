class SessionsController < ApplicationController
  skip_around_filter :require_thread_api_token, :only => [:destroy, :index]
  skip_around_filter :set_thread_api_token, :only => [:destroy, :index]
  skip_before_filter :find_object_by_uuid, :only => [:destroy, :index]

  def destroy
    session.clear
    redirect_to arvados_api_client.arvados_logout_url(return_to: root_url)
  end

  def index
    redirect_to root_url if session[:arvados_api_token]
    render_index
  end
end
