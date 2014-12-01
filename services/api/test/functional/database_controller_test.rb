require 'test_helper'

class DatabaseControllerTest < ActionController::TestCase
  include CurrentApiClient

  test "reset fails with non-admin token" do
    authorize_with :active
    post :reset
    assert_response 403
  end

  test "route not found when not in test mode" do
    authorize_with :admin
    env_was = Rails.env
    begin
      Rails.env = 'production'
      Rails.application.reload_routes!
      assert_raises ActionController::RoutingError do
        post :reset
      end
    ensure
      Rails.env = env_was
      Rails.application.reload_routes!
    end
  end

  test "reset fails when a non-test-fixture user exists" do
    act_as_system_user do
      User.create!(uuid: 'abcde-tpzed-123451234512345', email: 'bar@example.net')
    end
    authorize_with :admin
    post :reset
    assert_response 403
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
