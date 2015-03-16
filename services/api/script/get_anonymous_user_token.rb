#!/usr/bin/env ruby

# Get or Create an anonymous user token.
# If get option is used, an existing anonymous user token is returned. If none exist, one is created.
# If the get option is omitted, a new token is created and returned.

require 'trollop'

opts = Trollop::options do
  banner ''
  banner "Usage: get_anonymous_user_token "
  banner ''
  opt :get, <<-eos
Get an existing anonymous user token. If no such token exists \
or if this option is omitted, a new token is created and returned.
  eos
end

get_existing = opts[:get]

require File.dirname(__FILE__) + '/../config/environment'

include ApplicationHelper
include DbCurrentTime

act_as_system_user

def create_api_client_auth
  api_client_auth = ApiClientAuthorization.
    new(user: anonymous_user,
        api_client_id: 0,
        expires_at: db_current_time + 100.years,
        scopes: ['GET /'])
  api_client_auth.save!
  api_client_auth.reload
end

if get_existing
  api_client_auth = ApiClientAuthorization.
    where('user_id=?', anonymous_user.id.to_i).
    where('expires_at>?', db_current_time).
    select { |auth| auth.scopes == ['GET /'] }.
    first
end

# either not a get or no api_client_auth was found
if !api_client_auth
  api_client_auth = create_api_client_auth
end

# print it to the console
puts api_client_auth.api_token
