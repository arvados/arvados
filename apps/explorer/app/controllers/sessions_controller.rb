class SessionsController < ApplicationController
  skip_around_filter :thread_with_api_token, :only => [:destroy, :index]
  skip_before_filter :find_object_by_uuid, :only => [:destroy, :index]
  def destroy
    session.clear
    redirect_to $orvos_api_client.orvos_logout_url(return_to: logged_out_url)
  end
  def index
    redirect_to root_url if session[:orvos_api_token]
  end
end
