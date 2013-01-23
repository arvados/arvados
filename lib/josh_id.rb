require 'omniauth-oauth2'
module OmniAuth
  module Strategies
    class JoshId < OmniAuth::Strategies::OAuth2

      CUSTOM_PROVIDER_URL = 'http://auth.clinicalfuture.com'
      #CUSTOM_PROVIDER_URL = 'http://auth.clinicalfuture.com:3001'

      option :client_options, {
        :site =>  CUSTOM_PROVIDER_URL,
        :authorize_url => "#{CUSTOM_PROVIDER_URL}/auth/josh_id/authorize",
        :access_token_url => "#{CUSTOM_PROVIDER_URL}/auth/josh_id/access_token"
      }

      uid { raw_info['id'] }

      info do
        {
          :first_name => raw_info['info']['first_name'],
          :last_name => raw_info['info']['last_name'],
          :email => raw_info['info']['email'],
          :identity_url => raw_info['info']['identity_url'],
        }
      end

      extra do
        {
          'raw_info' => raw_info
        }
      end
      
      def raw_info
        @raw_info ||= access_token.get("/auth/josh_id/user.json?oauth_token=#{access_token.token}").parsed
      end
    end 
  end
end
