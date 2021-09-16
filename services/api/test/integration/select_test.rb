# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class SelectTest < ActionDispatch::IntegrationTest
  test "should select just two columns" do
    get "/arvados/v1/links",
      params: {:format => :json, :select => ['uuid', 'link_class']},
      headers: auth(:active)
    assert_response :success
    assert_equal json_response['items'].count, json_response['items'].select { |i|
      i.count == 3 and i['uuid'] != nil and i['link_class'] != nil
    }.count
  end

  test "fewer distinct than total count" do
    get "/arvados/v1/links",
      params: {:format => :json, :select => ['link_class']},
      headers: auth(:active)
    assert_response :success
    distinct_unspecified = json_response['items']

    get "/arvados/v1/links",
      params: {:format => :json, :select => ['link_class'], :distinct => false},
      headers: auth(:active)
    assert_response :success
    distinct_false = json_response['items']

    get "/arvados/v1/links",
      params: {:format => :json, :select => ['link_class'], :distinct => true},
      headers: auth(:active)
    assert_response :success
    distinct = json_response['items']

    assert_operator(distinct.count, :<, distinct_false.count,
                    "distinct=true count should be less than distinct=false count")
    assert_equal(distinct_unspecified.count, distinct_false.count,
                    "distinct=false should be the default")
    assert_equal distinct_false.uniq.count, distinct.count
  end

  test "select with order" do
    get "/arvados/v1/links",
      params: {:format => :json, :select => ['uuid'], :order => ["uuid asc"]},
      headers: auth(:active)
    assert_response :success

    assert json_response['items'].length > 0

    p = ""
    json_response['items'].each do |i|
      assert i['uuid'] > p
      p = i['uuid']
    end
  end

  test "select with default order" do
    get "/arvados/v1/links",
      params: {format: :json, select: ['uuid']},
      headers: auth(:admin)
    assert_response :success
    uuids = json_response['items'].collect { |i| i['uuid'] }
    assert_equal uuids, uuids.sort
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
    get "/arvados/v1/links",
      params: {
        :format => :json,
        :select => ['link_class', 'uuid'], :order => ['link_class asc', "uuid desc"]
      },
      headers: auth(:active)
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
    get "/arvados/v1/links",
      params: {
        :format => :json,
        :select => ['link_class', 'uuid'], :order => 'link_class asc, uuid desc'
      },
      headers: auth(:active)
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
