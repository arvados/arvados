#!/usr/bin/env ruby

# Install the supplied string (or a randomly generated token, if none
# is given) as an API token that authenticates to the system user
# account.
#
# Print the token on stdout.

supplied_token = ARGV[0]

require File.dirname(__FILE__) + '/../config/boot'
require File.dirname(__FILE__) + '/../config/environment'

include ApplicationHelper
act_as_system_user

if supplied_token
  api_client_auth = ApiClientAuthorization.
    where(api_token: supplied_token).
    first
  if api_client_auth && !api_client_auth.user.uuid.match(/-000000000000000$/)
    raise ActiveRecord::RecordNotUnique("Token already exists but is not a superuser token.")
  end
end

if !api_client_auth
  api_client_auth = ApiClientAuthorization.
    new(user: system_user,
        api_client_id: 0,
        created_by_ip_address: '::1',
        api_token: supplied_token)
  api_client_auth.save!
end

puts api_client_auth.api_token
