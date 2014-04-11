require 'test_helper'

class Arvados::V1::LogsControllerTest < ActionController::TestCase
  test "non-admins can read their own logs" do
    authorize_with :active
    post :create, log: {summary: "test log"}
    assert_response :success
    uuid = JSON.parse(@response.body)['uuid']
    assert_not_nil uuid
    get :show, {id: uuid}
    assert_response(:success, "failed to load created log")
    assert_equal("test log", assigns(:object).summary,
                 "loaded wrong log after creation")
  end
end
