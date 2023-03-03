# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class Arvados::V1::ContainerRequestsControllerTest < ActionController::TestCase
  def minimal_cr
    {
      command: ['echo', 'hello'],
      container_image: 'arvados/apitestfixture:latest',
      output_path: 'test',
      runtime_constraints: {vcpus: 1, ram: 1}
    }
  end

  test 'create with scheduling parameters' do
    authorize_with :active

    sp = {'partitions' => ['test1', 'test2']}
    post :create, params: {
           container_request: minimal_cr.merge(scheduling_parameters: sp.dup, state: "Committed")
         }
    assert_response :success

    cr = JSON.parse(@response.body)
    assert_not_nil cr, 'Expected container request'
    assert_equal sp['partitions'], cr['scheduling_parameters']['partitions']
    assert_equal false, cr['scheduling_parameters']['preemptible']
    assert_equal false, cr['scheduling_parameters']['supervisor']
  end

  test 'create a-c-r should be supervisor' do
    authorize_with :active

    post :create, params: {
           container_request: minimal_cr.merge(command: ["arvados-cwl-runner", "my-workflow.cwl"], state: "Committed")
         }
    assert_response :success

    cr = JSON.parse(@response.body)
    assert_not_nil cr, 'Expected container request'
    assert_equal true, cr['scheduling_parameters']['supervisor']
  end

  test "secret_mounts not in #create responses" do
    authorize_with :active

    post :create, params: {
           container_request: minimal_cr.merge(
             secret_mounts: {'/foo' => {'kind' => 'json', 'content' => 'bar'}}),
         }
    assert_response :success

    resp = JSON.parse(@response.body)
    refute resp.has_key?('secret_mounts')

    req = ContainerRequest.where(uuid: resp['uuid']).first
    assert_equal 'bar', req.secret_mounts['/foo']['content']
  end

  test "update with secret_mounts" do
    authorize_with :active
    req = container_requests(:uncommitted)

    patch :update, params: {
            id: req.uuid,
            container_request: {
              secret_mounts: {'/foo' => {'kind' => 'json', 'content' => 'bar'}},
            },
          }
    assert_response :success

    resp = JSON.parse(@response.body)
    refute resp.has_key?('secret_mounts')

    req.reload
    assert_equal 'bar', req.secret_mounts['/foo']['content']
  end

  test "cancel with runtime_constraints and scheduling_params with default values" do
    authorize_with :active
    req = container_requests(:queued)

    patch :update, params: {
      id: req.uuid,
      container_request: {
        state: 'Final',
        priority: 0,
        runtime_constraints: {
          'vcpus' => 1,
          'ram' => 123,
          'keep_cache_ram' => 0,
        },
        scheduling_parameters: {
          "preemptible"=>false
        }
      },
    }
    assert_response :success
  end

  test "update without deleting secret_mounts" do
    authorize_with :active
    req = container_requests(:uncommitted)
    req.update_attributes!(secret_mounts: {'/foo' => {'kind' => 'json', 'content' => 'bar'}})

    patch :update, params: {
            id: req.uuid,
            container_request: {
              command: ['echo', 'test'],
            },
          }
    assert_response :success

    resp = JSON.parse(@response.body)
    refute resp.has_key?('secret_mounts')

    req.reload
    assert_equal 'bar', req.secret_mounts['/foo']['content']
  end

  test "runtime_token not in #create responses" do
    authorize_with :active

    post :create, params: {
           container_request: minimal_cr.merge(
             runtime_token: api_client_authorizations(:spectator).token)
         }
    assert_response :success

    resp = JSON.parse(@response.body)
    refute resp.has_key?('runtime_token')

    req = ContainerRequest.where(uuid: resp['uuid']).first
    assert_equal api_client_authorizations(:spectator).token, req.runtime_token
  end

  %w(Running Complete).each do |state|
    test "filter on container.state = #{state}" do
      authorize_with :active
      get :index, params: {
            filters: [['container.state', '=', state]],
          }
      assert_response :success
      assert_operator json_response['items'].length, :>, 0
      json_response['items'].each do |cr|
        assert_equal state, Container.find_by_uuid(cr['container_uuid']).state
      end
    end
  end

  test "filter on container success" do
    authorize_with :active
    get :index, params: {
          filters: [
            ['container.state', '=', 'Complete'],
            ['container.exit_code', '=', '0'],
          ],
        }
    assert_response :success
    assert_operator json_response['items'].length, :>, 0
    json_response['items'].each do |cr|
      assert_equal 'Complete', Container.find_by_uuid(cr['container_uuid']).state
      assert_equal 0, Container.find_by_uuid(cr['container_uuid']).exit_code
    end
  end

  test "filter on container subproperty runtime_status[foo] = bar" do
    ctr = containers(:running)
    act_as_system_user do
      ctr.update_attributes!(runtime_status: {foo: 'bar'})
    end
    authorize_with :active
    get :index, params: {
          filters: [
            ['container.runtime_status.foo', '=', 'bar'],
          ],
        }
    assert_response :success
    assert_equal [ctr.uuid], json_response['items'].collect { |cr| cr['container_uuid'] }.uniq
  end
end
