# This file is called omniauth_init.rb instead of omniauth.rb because
# older versions had site configuration in omniauth.rb.
#
# It must come after omniauth.rb in (lexical) load order.

if defined? CUSTOM_PROVIDER_URL
  Rails.logger.warn "Copying omniauth from globals in legacy config file."
  Rails.configuration.sso_app_id = APP_ID
  Rails.configuration.sso_app_secret = APP_SECRET
  Rails.configuration.sso_provider_url = CUSTOM_PROVIDER_URL
else
  Rails.application.config.middleware.use OmniAuth::Builder do
    provider(:josh_id,
             Rails.configuration.sso_app_id,
             Rails.configuration.sso_app_secret,
             Rails.configuration.sso_provider_url)
  end
  OmniAuth.config.on_failure = StaticController.action(:login_failure)
end
