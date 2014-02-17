require 'test_helper'

class ApiClientAuthorizationsApiTest < ActionDispatch::IntegrationTest
  fixtures :all

  test "create system auth" do
    post "/arvados/v1/api_client_authorizations/create_system_auth", {:format => :json, :scopes => ['test'].to_json}, {'HTTP_AUTHORIZATION' => "OAuth2 #{api_client_authorizations(:admin_trustedclient).api_token}"}
    assert_response :success
  end

end
