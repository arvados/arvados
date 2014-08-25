require 'test_helper'

class SelectTest < ActionDispatch::IntegrationTest
  test "should select just two columns" do
    get "/arvados/v1/links", {:format => :json, :select => ['uuid', 'link_class']}, auth(:active)
    assert_response :success
    assert_equal json_response['items'].count, json_response['items'].select { |i|
      i.count == 3 and i['uuid'] != nil and i['link_class'] != nil
    }.count
  end

  test "fewer distinct than total count" do
    get "/arvados/v1/links", {:format => :json, :select => ['link_class'], :distinct => false}, auth(:active)
    assert_response :success
    links = json_response['items']

    get "/arvados/v1/links", {:format => :json, :select => ['link_class'], :distinct => true}, auth(:active)
    assert_response :success
    distinct = json_response['items']

    assert_operator(distinct.count, :<, links.count,
                    "distinct count should be less than link count")
    assert_equal links.uniq.count, distinct.count
  end

  test "select with order" do
    get "/arvados/v1/links", {:format => :json, :select => ['uuid'], :order => ["uuid asc"]}, auth(:active)
    assert_response :success

    assert json_response['items'].length > 0

    p = ""
    json_response['items'].each do |i|
      assert i['uuid'] > p
      p = i['uuid']
    end
  end

  def assert_link_classes_ascend(current_class, prev_class)
    # Databases and Ruby don't always agree about string ordering with
    # punctuation.  If the strings aren't ascending normally, check
    # that they're equal up to punctuation.
    if current_class < prev_class
      class_prefix = current_class.split(/\W/).first
      assert prev_class.start_with?(class_prefix)
    end
  end

  test "select two columns with order" do
    get "/arvados/v1/links", {:format => :json, :select => ['link_class', 'uuid'], :order => ['link_class asc', "uuid desc"]}, auth(:active)
    assert_response :success

    assert json_response['items'].length > 0

    prev_link_class = ""
    prev_uuid = "zzzzz-zzzzz-zzzzzzzzzzzzzzz"

    json_response['items'].each do |i|
      if prev_link_class != i['link_class']
        prev_uuid = "zzzzz-zzzzz-zzzzzzzzzzzzzzz"
      end

      assert_link_classes_ascend(i['link_class'], prev_link_class)
      assert i['uuid'] < prev_uuid

      prev_link_class = i['link_class']
      prev_uuid = i['uuid']
    end
  end

  test "select two columns with old-style order syntax" do
    get "/arvados/v1/links", {:format => :json, :select => ['link_class', 'uuid'], :order => 'link_class asc, uuid desc'}, auth(:active)
    assert_response :success

    assert json_response['items'].length > 0

    prev_link_class = ""
    prev_uuid = "zzzzz-zzzzz-zzzzzzzzzzzzzzz"

    json_response['items'].each do |i|
      if prev_link_class != i['link_class']
        prev_uuid = "zzzzz-zzzzz-zzzzzzzzzzzzzzz"
      end

      assert_link_classes_ascend(i['link_class'], prev_link_class)
      assert i['uuid'] < prev_uuid

      prev_link_class = i['link_class']
      prev_uuid = i['uuid']
    end
  end

end
