require 'test_helper'

class Arvados::V1::LogsControllerTest < ActionController::TestCase
  fixtures :logs

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

  test "test can still use where object_kind" do
    authorize_with :admin
    get :index, {
      where: { object_kind: 'arvados#user' }
    }
    assert_response :success
    found = assigns(:objects)
    assert_not_equal 0, found.count
    assert_equal found.count, (found.select { |f| f.object_uuid.match /[a-z0-9]{5}-tpzed-[a-z0-9]{15}/}).count
    l = JSON.parse(@response.body)
    assert_equal 'arvados#user', l['items'][0]['object_kind']
  end

  test "test can still use filter object_kind" do
    authorize_with :admin
    get :index, {
      filters: [ ['object_kind', '=', 'arvados#user'] ]
    }
    assert_response :success
    found = assigns(:objects)
    assert_not_equal 0, found.count
    assert_equal found.count, (found.select { |f| f.object_uuid.match /[a-z0-9]{5}-tpzed-[a-z0-9]{15}/}).count
  end

end
