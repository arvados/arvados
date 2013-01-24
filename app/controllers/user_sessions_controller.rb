class UserSessionsController < ApplicationController
  before_filter :login_required, :only => [ :destroy ]

  skip_before_filter :uncamelcase_params_hash_keys
  skip_before_filter :find_object_by_uuid

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

    # Give the authenticated user a cookie for direct API access
    session[:user_id] = user.id
    session[:user_uuid] = user.uuid
    session[:api_client_uuid] = nil
    session[:api_client_trusted] = true # full permission to see user's secrets

    @redirect_to = root_path
    if session.has_key? :return_to
      return send_api_token_to(session.delete :return_to)
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
    if current_user and params[:return_to]
      # Already logged in; just need to send a token to the requesting
      # API client.
      #
      # FIXME: if current_user has never authorized this app before,
      # ask for confirmation here!

      send_api_token_to(params[:return_to])
    else
      # TODO: make joshid propagate return_to as a GET parameter, and
      # use that GET parameter instead of session[] when redirecting
      # in create().  Using session[] is inappropriate: completing a
      # login in browser window A can cause a token to be sent to a
      # different API client who has requested a token in window B.

      session[:return_to] = params[:return_to]
      redirect_to "/auth/joshid"
    end
  end

  def send_api_token_to(callback_url)
    # Give the API client a token for making API calls on behalf of
    # the authenticated user

    # Stub: automatically register all new API clients
    api_client_url_prefix = callback_url.match(%r{^.*?://[^/]+})[0] + '/'
    api_client = ApiClient.find_or_create_by_url_prefix(api_client_url_prefix)

    api_client_auth = ApiClientAuthorization.
      new(user: user,
          api_client: api_client,
          created_by_ip_address: Thread.current[:remote_ip])
    api_client_auth.save!

    if callback_url.index('?')
      callback_url << '&'
    else
      callback_url << '?'
    end
    callback_url << 'api_token=' << api_client_auth.api_token
    redirect_to callback_url
  end
end
