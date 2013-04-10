require 'omniauth-oauth2'
module OmniAuth
  module Strategies
    class JoshId < OmniAuth::Strategies::OAuth2

      args [:client_id, :client_secret, :custom_provider_url]

      option :custom_provider_url, ''

      uid { raw_info['id'] }

      option :client_options, {}

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
      
      def client
        options.client_options[:site] = options[:custom_provider_url]
        options.client_options[:authorize_url] = "#{options[:custom_provider_url]}/auth/josh_id/authorize"
        options.client_options[:access_token_url] = "#{options[:custom_provider_url]}/auth/josh_id/access_token"
        ::OAuth2::Client.new(options.client_id, options.client_secret, deep_symbolize(options.client_options))
      end

      def callback_url
        full_host + script_name + callback_path + query_string
      end

      def raw_info
        @raw_info ||= access_token.get("/auth/josh_id/user.json?oauth_token=#{access_token.token}").parsed
      end
    end 
  end
end
