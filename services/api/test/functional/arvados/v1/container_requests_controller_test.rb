# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class Arvados::V1::ContainerRequestsControllerTest < ActionController::TestCase
  def minimal_cr
    {
      command: ['echo', 'hello'],
      container_image: 'test',
      output_path: 'test',
    }
  end

  test 'create with scheduling parameters' do
    authorize_with :active

    sp = {'partitions' => ['test1', 'test2']}
    post :create, {
           container_request: minimal_cr.merge(scheduling_parameters: sp.dup)
         }
    assert_response :success

    cr = JSON.parse(@response.body)
    assert_not_nil cr, 'Expected container request'
    assert_equal sp, cr['scheduling_parameters']
  end

  test "secret_mounts not in #create responses" do
    authorize_with :active

    post :create, {
           container_request: minimal_cr.merge(
             secret_mounts: {'/foo' => {'type' => 'json', 'content' => 'bar'}}),
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

    patch :update, {
            id: req.uuid,
            container_request: {
              secret_mounts: {'/foo' => {'type' => 'json', 'content' => 'bar'}},
            },
          }
    assert_response :success

    resp = JSON.parse(@response.body)
    refute resp.has_key?('secret_mounts')

    req.reload
    assert_equal 'bar', req.secret_mounts['/foo']['content']
  end

  test "update without deleting secret_mounts" do
    authorize_with :active
    req = container_requests(:uncommitted)
    req.update_attributes!(secret_mounts: {'/foo' => {'type' => 'json', 'content' => 'bar'}})

    patch :update, {
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
end
