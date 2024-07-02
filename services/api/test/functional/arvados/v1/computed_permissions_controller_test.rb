# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class Arvados::V1::ComputedPermissionsControllerTest < ActionController::TestCase
  test "require auth" do
    get :index, params: {}
    assert_response 401
  end

  test "require admin" do
    authorize_with :active
    get :index, params: {}
    assert_response 403
  end

  test "index with no options" do
    authorize_with :admin
    get :index, params: {}
    assert_response :success
    assert_operator 0, :<, json_response['items'].length

    last_user = ''
    last_target = ''
    json_response['items'].each do |item|
      assert_not_empty item['user_uuid']
      assert_not_empty item['target_uuid']
      assert_not_empty item['perm_level']
      # check default ordering
      assert_operator last_user, :<=, item['user_uuid']
      if last_user == item['user_uuid']
        assert_operator last_target, :<=, item['target_uuid']
      end
      last_user = item['user_uuid']
      last_target = item['target_uuid']
    end
  end

  test "index with limit" do
    authorize_with :admin
    get :index, params: {limit: 10}
    assert_response :success
    assert_equal 10, json_response['items'].length
  end

  test "index with filter on user_uuid" do
    user_uuid = users(:active).uuid
    authorize_with :admin
    get :index, params: {filters: [['user_uuid', '=', user_uuid]]}
    assert_response :success
    assert_not_equal 0, json_response['items'].length
    json_response['items'].each do |item|
      assert_equal user_uuid, item['user_uuid']
    end
  end

  test "index with filter on user_uuid and target_uuid" do
    user_uuid = users(:active).uuid
    target_uuid = groups(:aproject).uuid
    authorize_with :admin
    get :index, params: {filters: [
                           ['user_uuid', '=', user_uuid],
                           ['target_uuid', '=', target_uuid],
                         ]}
    assert_response :success
    assert_equal([{"user_uuid" => user_uuid,
                   "target_uuid" => target_uuid,
                   "perm_level" => "can_manage",
                  }],
                 json_response['items'])
  end

  test "index with disallowed filters" do
    authorize_with :admin
    get :index, params: {filters: [['perm_level', '=', 'can_manage']]}
    assert_response 422
  end

  %w(user_uuid target_uuid perm_level).each do |attr|
    test "select only #{attr}" do
      authorize_with :admin
      get :index, params: {select: [attr], limit: 1}
      assert_response :success
      assert_operator 0, :<, json_response['items'][0][attr].length
      assert_equal([{attr => json_response['items'][0][attr]}], json_response['items'])
    end
  end
end
