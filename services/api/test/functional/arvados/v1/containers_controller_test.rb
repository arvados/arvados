# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class Arvados::V1::ContainersControllerTest < ActionController::TestCase
  test 'create' do
    authorize_with :system_user
    post :create, params: {
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
      get :auth, params: {id: containers(:queued).uuid}
      assert_response 403
    end
  end

  test 'cannot get auth with wrong token' do
    authorize_with :dispatch1
    c = containers(:queued)
    assert c.lock, show_errors(c)

    authorize_with :system_user
    get :auth, params: {id: c.uuid}
    assert_response 403
  end

  test 'get auth' do
    authorize_with :dispatch1
    c = containers(:queued)
    assert c.lock, show_errors(c)
    get :auth, params: {id: c.uuid}
    assert_response :success
    assert_operator 32, :<, json_response['api_token'].length
    assert_equal 'arvados#apiClientAuthorization', json_response['kind']
  end

  test 'no auth or secret_mounts in container response' do
    authorize_with :dispatch1
    c = containers(:queued)
    assert c.lock, show_errors(c)
    get :show, params: {id: c.uuid}
    assert_response :success
    assert_nil json_response['auth']
    assert_nil json_response['secret_mounts']
  end

  test "lock container" do
    authorize_with :dispatch1
    uuid = containers(:queued).uuid
    post :lock, params: {id: uuid}
    assert_response :success
    assert_nil json_response['mounts']
    assert_nil json_response['command']
    assert_not_nil json_response['auth_uuid']
    assert_not_nil json_response['locked_by_uuid']
    assert_equal containers(:queued).uuid, json_response['uuid']
    assert_equal 'Locked', json_response['state']
    assert_equal containers(:queued).priority, json_response['priority']

    container = Container.where(uuid: uuid).first
    assert_equal 'Locked', container.state
    assert_not_nil container.locked_by_uuid
    assert_not_nil container.auth_uuid
  end

  test "unlock container" do
    authorize_with :dispatch1
    uuid = containers(:locked).uuid
    post :unlock, params: {id: uuid}
    assert_response :success
    assert_nil json_response['mounts']
    assert_nil json_response['command']
    assert_nil json_response['auth_uuid']
    assert_nil json_response['locked_by_uuid']
    assert_equal containers(:locked).uuid, json_response['uuid']
    assert_equal 'Queued', json_response['state']
    assert_equal containers(:locked).priority, json_response['priority']

    container = Container.where(uuid: uuid).first
    assert_equal 'Queued', container.state
    assert_nil container.locked_by_uuid
    assert_nil container.auth_uuid
  end

  test "unlock container locked by different dispatcher" do
    authorize_with :dispatch2
    uuid = containers(:locked).uuid
    post :unlock, params: {id: uuid}
    assert_response 403
  end

  [
    [:queued, :lock, :success, 'Locked'],
    [:queued, :unlock, 422, 'Queued'],
    [:locked, :lock, 422, 'Locked'],
    [:running, :lock, 422, 'Running'],
    [:running, :unlock, 422, 'Running'],
  ].each do |fixture, action, response, state|
    test "state transitions from #{fixture} to #{action}" do
      authorize_with :dispatch1
      uuid = containers(fixture).uuid
      post action, params: {id: uuid}
      assert_response response
      assert_equal state, Container.where(uuid: uuid).first.state
    end
  end

  test 'get current container for token' do
    authorize_with :running_container_auth
    get :current
    assert_response :success
    assert_equal containers(:running).uuid, json_response['uuid']
  end

  test 'no container associated with token' do
    authorize_with :dispatch1
    get :current
    assert_response 404
  end

  test 'try get current container, no token' do
    get :current
    assert_response 401
  end

  [
    [true, :running_container_auth],
    [false, :dispatch2],
    [false, :admin],
    [false, :active],
  ].each do |expect_success, auth|
    test "get secret_mounts with #{auth} token" do
      authorize_with auth
      get :secret_mounts, params: {id: containers(:running).uuid}
      if expect_success
        assert_response :success
        assert_equal "42\n", json_response["secret_mounts"]["/secret/6x9"]["content"]
      else
        assert_response 403
      end
    end
  end

  test 'get runtime_token auth' do
    authorize_with :dispatch2
    c = containers(:runtime_token)
    get :auth, params: {id: c.uuid}
    assert_response :success
    assert_equal "v2/#{json_response['uuid']}/#{json_response['api_token']}", api_client_authorizations(:container_runtime_token).token
    assert_equal 'arvados#apiClientAuthorization', json_response['kind']
  end

  test 'update_priority' do
    ActiveRecord::Base.connection.execute "update containers set priority=0 where uuid='#{containers(:running).uuid}'"
    authorize_with :admin
    post :update_priority, params: {id: containers(:running).uuid}
    assert_response :success
    assert_not_equal 0, Container.find_by_uuid(containers(:running).uuid).priority
  end
end
