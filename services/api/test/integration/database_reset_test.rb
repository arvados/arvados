require 'test_helper'

class DatabaseResetTest < ActionDispatch::IntegrationTest
  teardown do
    restore_configuration
    # We made configuration changes here that affect routing.
    Rails.application.reload_routes!
  end

  test "reset fails when Rails.env != 'test'" do
    rails_env_was = Rails.env
    begin
      Rails.env = 'production'
      Rails.application.reload_routes!
      post '/database/reset', {}, auth(:admin)
      assert_response 404
    ensure
      Rails.env = rails_env_was
    end
  end

  test "reset fails with non-admin token" do
    post '/database/reset', {}, auth(:active)
    assert_response 403
  end
end
