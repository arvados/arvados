#!/usr/bin/env ruby
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

# Get or Create an anonymous user token.
# If get option is used, an existing anonymous user token is returned. If none exist, one is created.
# If the get option is omitted, a new token is created and returned.

require 'optimist'

opts = Optimist::options do
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
  supplied_token = Rails.configuration.Users["AnonymousUserToken"]

  if supplied_token.nil? or supplied_token.empty?
    puts "Users.AnonymousUserToken is empty.  Destroying tokens that belong to anonymous."
    # Token is empty.  Destroy any anonymous tokens.
    ApiClientAuthorization.where(user: anonymous_user).destroy_all
    return nil
  end

  attr = {user: anonymous_user,
          api_client_id: 0,
          scopes: ['GET /']}

  secret = supplied_token

  if supplied_token[0..2] == 'v2/'
    _, token_uuid, secret, optional = supplied_token.split('/')
    if token_uuid[0..4] != Rails.configuration.ClusterID
      # Belongs to a different cluster.
      puts supplied_token
      return nil
    end
    attr[:uuid] = token_uuid
  end

  attr[:api_token] = secret

  api_client_auth = ApiClientAuthorization.where(attr).first
  if !api_client_auth
    # The anonymous user token should never expire but we are not allowed to
    # set :expires_at to nil, so we set it to 1000 years in the future.
    attr[:expires_at] = Time.now + 1000.years
    api_client_auth = ApiClientAuthorization.create!(attr)
  end
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
if api_client_auth
  puts "v2/#{api_client_auth.uuid}/#{api_client_auth.api_token}"
end
