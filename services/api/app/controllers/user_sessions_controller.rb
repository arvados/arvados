class UserSessionsController < ApplicationController
  before_filter :require_auth_scope, :only => [ :destroy ]

  skip_before_filter :find_object_by_uuid
  skip_before_filter :render_404_if_no_object

  respond_to :html

  # omniauth callback method
  def create
    omniauth = env['omniauth.auth']

    identity_url_ok = (omniauth['info']['identity_url'].length > 0) rescue false
    unless identity_url_ok
      # Whoa. This should never happen.
      logger.error "UserSessionsController.create: omniauth object missing/invalid"
      logger.error "omniauth.pretty_inspect():\n\n#{omniauth.pretty_inspect()}"

      return redirect_to login_failure_url
    end

    user = User.find_by_identity_url(omniauth['info']['identity_url'])
    if not user
      # Check for permission to log in to an existing User record with
      # a different identity_url
      Link.where("link_class = ? and name = ? and tail_uuid = ? and head_uuid like ?",
                 'permission',
                 'can_login',
                 omniauth['info']['email'],
                 User.uuid_like_pattern).each do |link|
        if prefix = link.properties['identity_url_prefix']
          if prefix == omniauth['info']['identity_url'][0..prefix.size-1]
            user = User.find_by_uuid(link.head_uuid)
            break if user
          end
        end
      end
    end
    if not user
      # New user registration
      user = User.new(:email => omniauth['info']['email'],
                      :first_name => omniauth['info']['first_name'],
                      :last_name => omniauth['info']['last_name'],
                      :identity_url => omniauth['info']['identity_url'],
                      :is_active => Rails.configuration.new_users_are_active,
                      :owner_uuid => system_user_uuid)
      act_as_system_user do
        user.save or raise Exception.new(user.errors.messages)
      end
    else
      user.email = omniauth['info']['email']
      user.first_name = omniauth['info']['first_name']
      user.last_name = omniauth['info']['last_name']
      if user.identity_url.nil?
        # First login to a pre-activated account
        user.identity_url = omniauth['info']['identity_url']
      end
    end

    # For the benefit of functional and integration tests:
    @user = user

    # prevent ArvadosModel#before_create and _update from throwing
    # "unauthorized":
    Thread.current[:user] = user

    user.save or raise Exception.new(user.errors.messages)

    omniauth.delete('extra')

    # Give the authenticated user a cookie for direct API access
    session[:user_id] = user.id
    session[:api_client_uuid] = nil
    session[:api_client_trusted] = true # full permission to see user's secrets

    @redirect_to = root_path
    if params.has_key?(:return_to)
      return send_api_token_to(params[:return_to], user)
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
    return_to = params[:return_to] || root_url
    redirect_to "#{CUSTOM_PROVIDER_URL}/users/sign_out?redirect_uri=#{CGI.escape return_to}"
  end

  # login - Just bounce to /auth/joshid. The only purpose of this function is
  # to save the return_to parameter (if it exists; see the application
  # controller). /auth/joshid bypasses the application controller.
  def login
    auth_method = params[:auth_method] || "google"
    if current_user and params[:return_to]
      # Already logged in; just need to send a token to the requesting
      # API client.
      #
      # FIXME: if current_user has never authorized this app before,
      # ask for confirmation here!

      send_api_token_to(params[:return_to], current_user)
    elsif params[:return_to]
      redirect_to "/auth/joshid?return_to=#{CGI.escape(params[:return_to])}&auth_method=#{CGI.escape(auth_method)}"
    else
      redirect_to "/auth/joshid?auth_method=#{CGI.escape(auth_method)}"
    end
  end

  def send_api_token_to(callback_url, user)
    # Give the API client a token for making API calls on behalf of
    # the authenticated user

    # Stub: automatically register all new API clients
    api_client_url_prefix = callback_url.match(%r{^.*?://[^/]+})[0] + '/'
    act_as_system_user do
      @api_client = ApiClient.find_or_create_by_url_prefix api_client_url_prefix
    end

    api_client_auth = ApiClientAuthorization.
      new(user: user,
          api_client: @api_client,
          created_by_ip_address: remote_ip,
          scopes: ["all"])
    api_client_auth.save!

    if callback_url.index('?')
      callback_url += '&'
    else
      callback_url += '?'
    end
    callback_url += 'api_token=' + api_client_auth.api_token
    redirect_to callback_url
  end
end
