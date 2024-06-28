# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

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
        if !api_client_auth
          # fall through to create a token
        elsif !api_client_auth.user.uuid.match(/-000000000000000$/)
          raise "Token exists but is not a superuser token."
        elsif api_client_auth.scopes != ['all']
          raise "Token exists but has limited scope #{api_client_auth.scopes.inspect}."
        end
      end

      # need to create a token
      if !api_client_auth
        # Check if there is an unexpired superuser token
        api_client_auth =
          ApiClientAuthorization.
          where(user_id: system_user.id).
          where_serialized(:scopes, ['all']).
          where('(expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)').
          first

        # none exist; create one with the supplied token
        if !api_client_auth
          api_client_auth = ApiClientAuthorization.
            new(user: system_user,
              created_by_ip_address: '::1',
              api_token: supplied_token)
          api_client_auth.save!
        end
      end

      "v2/" + api_client_auth.uuid + "/" + api_client_auth.api_token
    end
  end
end
