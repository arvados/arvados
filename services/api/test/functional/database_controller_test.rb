require 'test_helper'

class DatabaseControllerTest < ActionController::TestCase
  include CurrentApiClient

  test "reset fails with non-admin token" do
    authorize_with :active
    post :reset
    assert_response 403
  end

  test "reset fails when not in test mode" do
    authorize_with :admin
    env_was = ENV['RAILS_ENV']
    ENV['RAILS_ENV'] = 'development'
    post :reset
    assert_response 403
    ENV['RAILS_ENV'] = env_was
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
