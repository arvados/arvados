require 'test_helper'

class CollectionsApiTest < ActionDispatch::IntegrationTest
  fixtures :all

  test "should get index" do
    get "/arvados/v1/collections", {:format => :json}, {'HTTP_AUTHORIZATION' => "OAuth2 #{api_client_authorizations(:active).api_token}"}
    assert_response :success
    assert_equal "arvados#collectionList", jresponse['kind']
  end

  test "controller 404 response is json" do
    get "/arvados/v1/thingsthatdonotexist", {:format => :xml}, {'HTTP_AUTHORIZATION' => "OAuth2 #{api_client_authorizations(:active).api_token}"}
    assert_response 404
    assert_equal 1, jresponse['errors'].length
    assert_equal true, jresponse['errors'][0].is_a?(String)
  end

  test "object 404 response is json" do
    get "/arvados/v1/groups/zzzzz-j7d0g-o5ba971173cup4f", {}, {'HTTP_AUTHORIZATION' => "OAuth2 #{api_client_authorizations(:active).api_token}"}
    assert_response 404
    assert_equal 1, jresponse['errors'].length
    assert_equal true, jresponse['errors'][0].is_a?(String)
  end

end
