require 'test_helper'

class DatabaseControllerTest < ActionController::TestCase
  include CurrentApiClient

  teardown do
    restore_configuration
    # We made configuration changes here that affect routing.
    Rails.application.reload_routes!
  end

  test "reset fails with non-admin token" do
    authorize_with :active
    post :reset
    assert_response 403
  end

  test "reset fails when not in test mode" do
    authorize_with :admin
    env_was = Rails.env
    begin
      Rails.env = 'development'
      post :reset
      assert_response 403
    ensure
      Rails.env = env_was
    end
  end

  test "reset fails when not configured" do
    Rails.configuration.enable_remote_database_reset = false
    Rails.application.reload_routes!
    authorize_with :admin
    assert_raise ActionController::RoutingError do
      post :reset
    end
  end

  test "reset succeeds with admin token" do
    new_uuid = nil
    act_as_system_user do
      new_uuid = Specimen.create.uuid
    end
    assert_not_empty Specimen.where(uuid: new_uuid)
    authorize_with :admin
    post :reset
    assert_response 200
    assert_empty Specimen.where(uuid: new_uuid)
  end
end
