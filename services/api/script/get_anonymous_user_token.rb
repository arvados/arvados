#!/usr/bin/env ruby
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

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
  opt :token, "token to create (optional)", :type => :string
end

get_existing = opts[:get]
supplied_token = opts[:token]

require File.dirname(__FILE__) + '/../config/environment'

include ApplicationHelper
act_as_system_user

def create_api_client_auth(supplied_token=nil)

  # If token is supplied, see if it exists
  if supplied_token
    api_client_auth = ApiClientAuthorization.
      where(api_token: supplied_token).
      first
    if !api_client_auth
      # fall through to create a token
    else
      raise "Token exists, aborting!"
    end
  end

  api_client_auth = ApiClientAuthorization.
    new(user: anonymous_user,
        api_client_id: 0,
        expires_at: Time.now + 100.years,
        scopes: ['GET /'],
        api_token: supplied_token)
  api_client_auth.save!
  api_client_auth.reload
  api_client_auth
end

if get_existing
  api_client_auth = ApiClientAuthorization.
    where('user_id=?', anonymous_user.id.to_i).
    where('expires_at>?', Time.now).
    select { |auth| auth.scopes == ['GET /'] }.
    first
end

# either not a get or no api_client_auth was found
if !api_client_auth
  api_client_auth = create_api_client_auth(supplied_token)
end

# print it to the console
puts api_client_auth.api_token
