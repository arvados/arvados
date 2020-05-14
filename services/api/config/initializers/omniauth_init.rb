# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

# This file is called omniauth_init.rb instead of omniauth.rb because
# older versions had site configuration in omniauth.rb.
#
# It must come after omniauth.rb in (lexical) load order.

if defined? CUSTOM_PROVIDER_URL
  Rails.logger.warn "Copying omniauth from globals in legacy config file."
  Rails.configuration.Login["SSO"]["ProviderAppID"] = APP_ID
  Rails.configuration.Login["SSO"]["ProviderAppSecret"] = APP_SECRET
  Rails.configuration.Services["SSO"]["ExternalURL"] = CUSTOM_PROVIDER_URL.sub(/\/$/, "") + "/"
else
  Rails.application.config.middleware.use OmniAuth::Builder do
    provider(:josh_id,
             Rails.configuration.Login["SSO"]["ProviderAppID"],
             Rails.configuration.Login["SSO"]["ProviderAppSecret"],
             Rails.configuration.Services["SSO"]["ExternalURL"])
  end
  OmniAuth.config.on_failure = StaticController.action(:login_failure)
end
