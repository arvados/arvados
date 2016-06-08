require 'test_helper'

class Arvados::V1::ContainersControllerTest < ActionController::TestCase
  test 'create' do
    authorize_with :system_user
    post :create, {
      container: {
        command: ['echo', 'hello'],
        container_image: 'test',
        output_path: 'test',
      },
    }
    assert_response :success
  end

  [Container::Queued, Container::Complete].each do |state|
    test "cannot get auth in #{state} state" do
      authorize_with :dispatch1
      get :auth, id: containers(:queued).uuid
      assert_response 403
    end
  end

  test 'cannot get auth with wrong token' do
    authorize_with :dispatch1
    c = containers(:queued)
    assert c.update_attributes(state: Container::Locked), show_errors(c)

    authorize_with :system_user
    get :auth, id: c.uuid
    assert_response 403
  end

  test 'get auth' do
    authorize_with :dispatch1
    c = containers(:queued)
    assert c.update_attributes(state: Container::Locked), show_errors(c)
    get :auth, id: c.uuid
    assert_response :success
    assert_operator 32, :<, json_response['api_token'].length
    assert_equal 'arvados#apiClientAuthorization', json_response['kind']
  end

  test 'no auth in container response' do
    authorize_with :dispatch1
    c = containers(:queued)
    assert c.update_attributes(state: Container::Locked), show_errors(c)
    get :show, id: c.uuid
    assert_response :success
    assert_nil json_response['auth']
  end
end
