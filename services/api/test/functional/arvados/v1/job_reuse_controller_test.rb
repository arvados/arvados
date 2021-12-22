# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'
require 'helpers/git_test_helper'

class Arvados::V1::JobReuseControllerTest < ActionController::TestCase
  fixtures :repositories, :users, :jobs, :links, :collections

  setup do
    @controller = Arvados::V1::JobsController.new
    authorize_with :active
  end

  BASE_FILTERS = {
    'repository' => ['=', 'active/foo'],
    'script' => ['=', 'hash'],
    'script_version' => ['in git', 'main'],
    'docker_image_locator' => ['=', nil],
    'arvados_sdk_version' => ['=', nil],
  }

  def filters_from_hash(hash)
    hash.each_pair.map { |name, filter| [name] + filter }
  end

  test "find Job with script version range" do
    get :index, params: {
      filters: [["repository", "=", "active/foo"],
                ["script", "=", "hash"],
                ["script_version", "in git", "tag1"]]
    }
    assert_response :success
    assert_not_nil assigns(:objects)
    assert_includes(assigns(:objects).map { |job| job.uuid },
                    jobs(:previous_job_run).uuid)
  end

  test "find Job with script version range exclusions" do
    get :index, params: {
      filters: [["repository", "=", "active/foo"],
                ["script", "=", "hash"],
                ["script_version", "not in git", "tag1"]]
    }
    assert_response :success
    assert_not_nil assigns(:objects)
    refute_includes(assigns(:objects).map { |job| job.uuid },
                    jobs(:previous_job_run).uuid)
  end

  test "find Job with Docker image range" do
    get :index, params: {
      filters: [["docker_image_locator", "in docker",
                 "arvados/apitestfixture"]]
    }
    assert_response :success
    assert_not_nil assigns(:objects)
    assert_includes(assigns(:objects).map { |job| job.uuid },
                    jobs(:previous_docker_job_run).uuid)
    refute_includes(assigns(:objects).map { |job| job.uuid },
                    jobs(:previous_job_run).uuid)
  end

  test "find Job with Docker image using reader tokens" do
    authorize_with :inactive
    get(:index, params: {
          filters: [["docker_image_locator", "in docker",
                     "arvados/apitestfixture"]],
          reader_tokens: [api_token(:active)],
        })
    assert_response :success
    assert_not_nil assigns(:objects)
    assert_includes(assigns(:objects).map { |job| job.uuid },
                    jobs(:previous_docker_job_run).uuid)
    refute_includes(assigns(:objects).map { |job| job.uuid },
                    jobs(:previous_job_run).uuid)
  end

  test "'in docker' filter accepts arrays" do
    get :index, params: {
      filters: [["docker_image_locator", "in docker",
                ["_nonesuchname_", "arvados/apitestfixture"]]]
    }
    assert_response :success
    assert_not_nil assigns(:objects)
    assert_includes(assigns(:objects).map { |job| job.uuid },
                    jobs(:previous_docker_job_run).uuid)
    refute_includes(assigns(:objects).map { |job| job.uuid },
                    jobs(:previous_job_run).uuid)
  end

  test "'not in docker' filter accepts arrays" do
    get :index, params: {
      filters: [["docker_image_locator", "not in docker",
                ["_nonesuchname_", "arvados/apitestfixture"]]]
    }
    assert_response :success
    assert_not_nil assigns(:objects)
    assert_includes(assigns(:objects).map { |job| job.uuid },
                    jobs(:previous_job_run).uuid)
    refute_includes(assigns(:objects).map { |job| job.uuid },
                    jobs(:previous_docker_job_run).uuid)
  end

end
