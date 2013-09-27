require 'test_helper'

class CollectionsApiTest < ActionDispatch::IntegrationTest
  fixtures :all

  test "should get index" do
    get "/arvados/v1/collections", {:format => :json}, {'HTTP_AUTHORIZATION' => "OAuth2 #{api_client_authorizations(:active).api_token}"}
    @json_response ||= ActiveSupport::JSON.decode @response.body
    assert_response :success
    assert_equal "arvados#collectionList", @json_response['kind']
  end

end
