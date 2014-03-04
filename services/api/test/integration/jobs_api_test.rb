require 'test_helper'

class JobsApiTest < ActionDispatch::IntegrationTest
  fixtures :all

  test "cancel job" do
    post "/arvados/v1/jobs/#{jobs(:running).uuid}/cancel", {:format => :json}, {'HTTP_AUTHORIZATION' => "OAuth2 #{api_client_authorizations(:active).api_token}"}
    assert_response :success
    assert_equal "arvados#job", jresponse['kind']
    assert_not_nil jresponse['cancelled_at']
  end

  test "cancel someone else's visible job" do
    post "/arvados/v1/jobs/#{jobs(:barbaz).uuid}/cancel", {:format => :json}, {'HTTP_AUTHORIZATION' => "OAuth2 #{api_client_authorizations(:spectator).api_token}"}
    assert_response 403
  end

  test "cancel someone else's invisible job" do
    post "/arvados/v1/jobs/#{jobs(:running).uuid}/cancel", {:format => :json}, {'HTTP_AUTHORIZATION' => "OAuth2 #{api_client_authorizations(:spectator).api_token}"}
    assert_response 404
  end

end
