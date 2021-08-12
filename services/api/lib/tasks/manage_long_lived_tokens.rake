# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

# Tasks that can be useful when changing token expiration policies by assigning
# a non-zero value to Login.TokenLifetime config.

require 'set'
require 'current_api_client'

namespace :db do
  desc "Apply expiration policy on long lived tokens"
  task fix_long_lived_tokens: :environment do
    lifetime = Rails.configuration.API.MaxTokenLifetime
    if lifetime.nil? or lifetime == 0
      lifetime = Rails.configuration.Login.TokenLifetime
    end
    if lifetime.nil? or lifetime == 0
      puts("No expiration policy set (API.MaxTokenLifetime nor Login.TokenLifetime is set), nothing to do.")
      next
    end
    exp_date = Time.now + lifetime
    puts("Setting token expiration to: #{exp_date}")
    token_count = 0
    ll_tokens(lifetime).each do |auth|
      if auth.user.nil?
        printf("*** WARNING, found ApiClientAuthorization with invalid user: auth id: %d, user id: %d\n", auth.id, auth.user_id)
        next
      end
      if (auth.user.uuid =~ /-tpzed-000000000000000/).nil?
        CurrentApiClientHelper.act_as_system_user do
          auth.update_attributes!(expires_at: exp_date)
        end
        token_count += 1
      end
    end
    puts("#{token_count} tokens updated.")
  end

  desc "Show users with long lived tokens"
  task check_long_lived_tokens: :environment do
    lifetime = Rails.configuration.API.MaxTokenLifetime
    if lifetime.nil? or lifetime == 0
      lifetime = Rails.configuration.Login.TokenLifetime
    end
    if lifetime.nil? or lifetime == 0
      puts("No expiration policy set (API.MaxTokenLifetime nor Login.TokenLifetime is set), nothing to do.")
      next
    end
    user_ids = Set.new()
    token_count = 0
    ll_tokens(lifetime).each do |auth|
      if auth.user.nil?
        printf("*** WARNING, found ApiClientAuthorization with invalid user: auth id: %d, user id: %d\n", auth.id, auth.user_id)
        next
      end
      if not auth.user.nil? and (auth.user.uuid =~ /-tpzed-000000000000000/).nil?
        user_ids.add(auth.user_id)
        token_count += 1
      end
    end

    if user_ids.size > 0
      puts("Found #{token_count} long-lived tokens from users:")
      user_ids.each do |uid|
        u = User.find(uid)
        puts("#{u.username},#{u.email},#{u.uuid}") if !u.nil?
      end
    else
      puts("No long-lived tokens found.")
    end
  end

  def ll_tokens(lifetime)
    query = ApiClientAuthorization.where(expires_at: nil)
    query = query.or(ApiClientAuthorization.where("expires_at > ?", Time.now + lifetime))
    query
  end
end
