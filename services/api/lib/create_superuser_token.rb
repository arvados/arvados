# Install the supplied string (or a randomly generated token, if none
# is given) as an API token that authenticates to the system user account.

module CreateSuperUserToken
  require File.dirname(__FILE__) + '/../config/boot'
  require File.dirname(__FILE__) + '/../config/environment'

  include ApplicationHelper

  def create_superuser_token supplied_token=nil
    act_as_system_user do
      # If token is supplied, verify that it indeed is a superuser token
      if supplied_token
        api_client_auth = ApiClientAuthorization.
          where(api_token: supplied_token).
          first
        if api_client_auth && !api_client_auth.user.uuid.match(/-000000000000000$/)
          raise "Token already exists but is not a superuser token."
        end
      end

      # need to create a token
      if !api_client_auth
        # Get (or create) trusted api client
        apiClient =  ApiClient.find_or_create_by_url_prefix_and_is_trusted("ssh://root@localhost/", true)

        # Check if there is an unexpired superuser token corresponding to this api client
        api_client_auth = ApiClientAuthorization.where(
                'user_id = (?) AND
                 api_client_id = (?) AND
                 (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)',
               system_user.id, apiClient.id).first

        # none exist; create one with the supplied token
        if !api_client_auth
          api_client_auth = ApiClientAuthorization.
            new(user: system_user,
              api_client_id: apiClient.id,
              created_by_ip_address: '::1',
              api_token: supplied_token)
          api_client_auth.save!
        end
      end

      api_client_auth.api_token
    end
  end
end
