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

  test "lock and unlock container" do
    # lock container
    authorize_with :dispatch1
    post :lock, {id: containers(:queued).uuid}
    assert_response :success
    container = Container.where(uuid: containers(:queued).uuid).first
    assert_equal 'Locked', container.state
    assert_not_nil container.locked_by_uuid
    assert_not_nil container.auth_uuid

    # unlock container
    @test_counter = 0  # Reset executed action counter
    @controller = Arvados::V1::ContainersController.new
    authorize_with :dispatch1
    post :unlock, {id: container.uuid}
    assert_response :success
    container = Container.where(uuid: container.uuid).first
    assert_equal 'Queued', container.state
    assert_nil container.locked_by_uuid
    assert_nil container.auth_uuid
  end

  def create_new_container attrs={}
    attrs = {
      command: ['echo', 'foo'],
      container_image: 'img',
      output_path: '/tmp',
      priority: 1,
      runtime_constraints: {"vcpus" => 1, "ram" => 1},
    }
    c = Container.new attrs.merge(attrs)
    c.save!
    cr = ContainerRequest.new attrs.merge(attrs)
    cr.save!
    assert cr.update_attributes(container_uuid: c.uuid,
                                state: ContainerRequest::Committed,
                               ), show_errors(cr)

    return c
  end

  [
    [['Queued', :success], ['Locked', :success]],
    [['Queued', :success], ['Locked', :success], ['Locked', 422]],
    [['Queued', :success], ['Locked', :success], ['Queued', :success]],
    [['Queued', :success], ['Locked', :success], ['Running', :success], ['Queued', 422]],
  ].each do |transitions|
    test "lock and unlock state transitions #{transitions}" do
      authorize_with :dispatch1

      container = create_new_container()

      transitions.each do |state, status|
        @test_counter = 0  # Reset executed action counter
        @controller = Arvados::V1::ContainersController.new
        authorize_with :dispatch1

        if state == 'Locked'
          post :lock, {id: container.uuid}
        elsif state == 'Queued'
          post :unlock, {id: container.uuid}
        else
          container.update_attributes!(state: state)
        end
        assert_response status

        container = Container.where(uuid: container['uuid']).first
        assert_equal state, container.state if status == :success
      end
    end
  end
end
