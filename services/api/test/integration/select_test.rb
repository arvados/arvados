require 'test_helper'

class SelectTest < ActionDispatch::IntegrationTest
  test "should select just two columns" do
    get "/arvados/v1/links", {:format => :json, :select => ['uuid', 'link_class']}, auth(:active)
    assert_response :success
    assert_equal json_response['items'].count, json_response['items'].select { |i|
      i['uuid'] != nil and i['link_class'] != nil and i['head_uuid'] == nil and i['tail_uuid'] == nil
    }.count
  end

end
