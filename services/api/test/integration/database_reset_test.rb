require 'test_helper'

class DatabaseResetTest < ActionDispatch::IntegrationTest
  teardown do
    restore_configuration
    # We made configuration changes here that affect routing.
    Rails.application.reload_routes!
  end

  test "reset fails when not configured" do
    Rails.configuration.enable_remote_database_reset = false
    Rails.application.reload_routes!
    post '/database/reset', {}, auth(:admin)
    assert_response 404
  end

  test "reset fails with non-admin token" do
    post '/database/reset', {}, auth(:active)
    assert_response 403
  end
end
