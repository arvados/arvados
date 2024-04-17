# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class DatabaseResetTest < ActionDispatch::IntegrationTest
  slow_test "reset fails when Rails.env != 'test'" do
    rails_env_was = Rails.env
    begin
      Rails.env = 'production'
      Rails.application.reload_routes!
      post '/database/reset', params: {}, headers: auth(:admin)
      assert_response 404
    ensure
      Rails.env = rails_env_was
      Rails.application.reload_routes!
    end
  end

  test "reset fails with non-admin token" do
    post '/database/reset', params: {}, headers: auth(:active)
    assert_response 403
  end

  slow_test "database reset doesn't break basic CRUD operations" do
    active_auth = auth(:active)
    admin_auth = auth(:admin)

    authorize_with :admin
    post '/database/reset', params: {}, headers: admin_auth
    assert_response :success

    post '/arvados/v1/collections', params: {collection: '{}'}, headers: active_auth
    assert_response :success
    new_uuid = json_response['uuid']

    get '/arvados/v1/collections/'+new_uuid, params: {}, headers: active_auth
    assert_response :success

    put('/arvados/v1/collections/'+new_uuid,
      params: {collection: '{"properties":{}}'},
      headers: active_auth)
    assert_response :success

    delete '/arvados/v1/collections/'+new_uuid, params: {}, headers: active_auth
    assert_response :success

    get '/arvados/v1/collections/'+new_uuid, params: {}, headers: active_auth
    assert_response 404
  end

  slow_test "roll back database change" do
    active_auth = auth(:active)
    admin_auth = auth(:admin)

    old_uuid = collections(:collection_owned_by_active).uuid
    authorize_with :admin
    post '/database/reset', params: {}, headers: admin_auth
    assert_response :success

    delete '/arvados/v1/collections/' + old_uuid, params: {}, headers: active_auth
    assert_response :success
    post '/arvados/v1/collections', params: {collection: '{}'}, headers: active_auth
    assert_response :success
    new_uuid = json_response['uuid']

    # Reset to fixtures.
    post '/database/reset', params: {}, headers: admin_auth
    assert_response :success

    # New collection should disappear. Old collection should reappear.
    get '/arvados/v1/collections/'+new_uuid, params: {}, headers: active_auth
    assert_response 404
    get '/arvados/v1/collections/'+old_uuid, params: {}, headers: active_auth
    assert_response :success
  end
end
