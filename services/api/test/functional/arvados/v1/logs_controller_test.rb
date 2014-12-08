require 'test_helper'

class Arvados::V1::LogsControllerTest < ActionController::TestCase
  fixtures :logs

  test "non-admins can create their own logs" do
    authorize_with :active
    post :create, log: {summary: 'test log'}
    assert_response :success
    resp = assigns(:object)
    assert_not_nil resp.uuid
    assert_equal('test log', resp.summary, "loaded wrong log after creation")
  end

  test "non-admins can read their own logs" do
    authorize_with :active
    my_log = logs(:log_owned_by_active)
    get :show, {id: my_log[:uuid]}
    assert_response(:success, "failed to get log")
    resp = assigns(:object)
    assert_equal(my_log[:summary], resp.summary, "got wrong log")
  end

  test "test can still use where object_kind" do
    authorize_with :admin
    get :index, {
      where: { object_kind: 'arvados#user' }
    }
    assert_response :success
    found = assigns(:objects)
    assert_not_equal 0, found.count
    assert_equal found.count, (found.select { |f| f.object_uuid.match User.uuid_regex }).count
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
    assert_equal found.count, (found.select { |f| f.object_uuid.match User.uuid_regex }).count
  end

end
