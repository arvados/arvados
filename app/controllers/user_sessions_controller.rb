class UserSessionsController < ApplicationController
  before_filter :login_required, :only => [ :destroy ]

  skip_before_filter :uncamelcase_params_hash_keys
  skip_before_filter :find_object_by_uuid
  skip_before_filter :authenticate_api_token

  respond_to :html

  # omniauth callback method
  def create
    omniauth = env['omniauth.auth']
    #logger.debug "+++ #{omniauth}"

    identity_url_ok = (omniauth['info']['identity_url'].length > 0) rescue false
    unless identity_url_ok
      # Whoa. This should never happen.

      @title = "UserSessionsController.create: omniauth object missing/invalid"
      @body = "omniauth.pretty_inspect():\n\n#{omniauth.pretty_inspect()}"

      view_context.fatal_error(@title,@body)
      return redirect_to openid_login_error_url
    end

    user = User.find_by_identity_url(omniauth['info']['identity_url'])
    if not user
      # New user registration
      user = User.create!(:email => omniauth['info']['email'],
                          :first_name => omniauth['info']['first_name'],
                          :last_name => omniauth['info']['last_name'],
                          :identity_url => omniauth['info']['identity_url'])
    else
      user.email = omniauth['info']['email']
      user.first_name = omniauth['info']['first_name']
      user.last_name = omniauth['info']['last_name']
      user.save
    end

    omniauth.delete('extra')

    session[:user_id] = user.id

    @redirect_to = root_path
    if session.has_key?('redirect_to') then
      @redirect_to = session[:redirect_to]
      session.delete(:redirect_to)
    end
    redirect_to @redirect_to
  end

  # Omniauth failure callback
  def failure
    flash[:notice] = params[:message]
  end

  # logout - Clear our rack session BUT essentially redirect to the provider
  # to clean up the Devise session from there too !
  def logout
    session[:user_id] = nil

    flash[:notice] = 'You have logged off'
    redirect_to "#{CUSTOM_PROVIDER_URL}/users/sign_out?redirect_uri=#{root_url}"
  end

  # login - Just bounce to /auth/joshid. The only purpose of this function is
  # to save the redirect_to parameter (if it exists; see the application
  # controller). /auth/joshid bypasses the application controller.
  def login
    redirect_to "/auth/joshid"
  end
end
