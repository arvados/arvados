require 'test_helper'

class LinksApiTest < ActionDispatch::IntegrationTest
  fixtures :all

  test "should get index" do
    get "/arvados/v1/links", {
      :where => '{"tail_kind":"arvados#user"}',
      :format => :json
    }, auth(:active)
    assert_response :success
    assert_equal "arvados#linkList", jresponse['kind']
  end

  test "get index with tail_kind filter" do
    get "/arvados/v1/links", {
      :filters => '[["tail_kind","=","arvados#user"]]',
      :format => :json
    }, auth(:active)
    assert_response :success
    assert_equal "arvados#linkList", jresponse['kind']
    jresponse['items'].each do |i|
      assert_equal 'arvados#user', i['tail_kind']
    end
  end

  test "get index with name and tail_uuid filter" do
    name_list = %w(can_manage can_read)
    get "/arvados/v1/links", {
      :filters => [
                   ["tail_uuid", "=", users(:active).uuid],
                   ["name", "in", name_list],
                  ].to_json,
      :format => :json
    }, auth(:active)
    assert_response :success
    assert_equal "arvados#linkList", jresponse['kind']
    jresponse['items'].each do |i|
      assert_equal 'arvados#user', i['tail_kind']
      assert_equal users(:active).uuid, i['tail_uuid']
      assert_not_nil name_list.index(i['name'])
    end
  end

  test "get index with tail_kind in filters[] and in where{}" do
    get "/arvados/v1/links", {
      :where => '{"tail_kind":"arvados#user"}',
      :filters => '[["tail_kind","=","arvados#user"]]',
      :format => :json
    }, auth(:active)
    assert_response :success
    assert_equal "arvados#linkList", jresponse['kind']
    jresponse['items'].each do |i|
      assert_equal 'arvados#user', i['tail_kind']
    end
  end
end
