require 'test_helper'

class DatabaseResetTest < ActionDispatch::IntegrationTest
  self.use_transactional_fixtures = false

  test "reset fails when Rails.env != 'test'" do
    rails_env_was = Rails.env
    begin
      Rails.env = 'production'
      Rails.application.reload_routes!
      post '/database/reset', {}, auth(:admin)
      assert_response 404
    ensure
      Rails.env = rails_env_was
      Rails.application.reload_routes!
    end
  end

  test "reset fails with non-admin token" do
    post '/database/reset', {}, auth(:active)
    assert_response 403
  end

  test "database reset doesn't break basic CRUD operations" do
    active_auth = auth(:active)
    admin_auth = auth(:admin)

    authorize_with :admin
    post '/database/reset', {}, admin_auth
    assert_response :success

    post '/arvados/v1/specimens', {specimen: '{}'}, active_auth
    assert_response :success
    new_uuid = json_response['uuid']

    get '/arvados/v1/specimens/'+new_uuid, {}, active_auth
    assert_response :success

    put('/arvados/v1/specimens/'+new_uuid,
        {specimen: '{"properties":{}}'}, active_auth)
    assert_response :success

    delete '/arvados/v1/specimens/'+new_uuid, {}, active_auth
    assert_response :success

    get '/arvados/v1/specimens/'+new_uuid, {}, active_auth
    assert_response 404
  end

  test "roll back database change" do
    active_auth = auth(:active)
    admin_auth = auth(:admin)

    old_uuid = specimens(:owned_by_active_user).uuid
    authorize_with :admin
    post '/database/reset', {}, admin_auth
    assert_response :success

    delete '/arvados/v1/specimens/' + old_uuid, {}, active_auth
    assert_response :success
    post '/arvados/v1/specimens', {specimen: '{}'}, active_auth
    assert_response :success
    new_uuid = json_response['uuid']

    # Reset to fixtures.
    post '/database/reset', {}, admin_auth
    assert_response :success

    # New specimen should disappear. Old specimen should reappear.
    get '/arvados/v1/specimens/'+new_uuid, {}, active_auth
    assert_response 404
    get '/arvados/v1/specimens/'+old_uuid, {}, active_auth
    assert_response :success
  end
end
