# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class Arvados::V1::SchemaControllerTest < ActionController::TestCase

  setup do forget end
  teardown do forget end
  def forget
    Rails.cache.delete 'arvados_v1_rest_discovery'
    AppVersion.forget
  end

  test "should get fresh discovery document" do
    MAX_SCHEMA_AGE = 60
    get :index
    assert_response :success
    discovery_doc = JSON.parse(@response.body)
    assert_equal 'discovery#restDescription', discovery_doc['kind']
    assert_equal(true,
                 Time.now - MAX_SCHEMA_AGE.seconds < discovery_doc['generatedAt'],
                 "discovery document was generated >#{MAX_SCHEMA_AGE}s ago")
  end

  test "discovery document fields" do
    get :index
    assert_response :success
    discovery_doc = JSON.parse(@response.body)
    assert_includes discovery_doc, 'defaultTrashLifetime'
    assert_equal discovery_doc['defaultTrashLifetime'], Rails.application.config.default_trash_lifetime
    assert_match(/^[0-9a-f]+(-modified)?$/, discovery_doc['source_version'])
    assert_equal discovery_doc['websocketUrl'], Rails.application.config.websocket_address
    assert_equal discovery_doc['workbenchUrl'], Rails.application.config.workbench_address
    assert_equal('zzzzz', discovery_doc['uuidPrefix'])
  end

  test "discovery document overrides source_version with config" do
    Rails.configuration.source_version = 'aaa888fff'
    get :index
    assert_response :success
    discovery_doc = JSON.parse(@response.body)
    assert_equal 'aaa888fff', discovery_doc['source_version']
  end

  test "empty disable_api_methods" do
    get :index
    assert_response :success
    discovery_doc = JSON.parse(@response.body)
    assert_equal('POST',
                 discovery_doc['resources']['jobs']['methods']['create']['httpMethod'])
  end

  test "non-empty disable_api_methods" do
    Rails.configuration.disable_api_methods =
      ['jobs.create', 'pipeline_instances.create', 'pipeline_templates.create']
    get :index
    assert_response :success
    discovery_doc = JSON.parse(@response.body)
    ['jobs', 'pipeline_instances', 'pipeline_templates'].each do |r|
      refute_includes(discovery_doc['resources'][r]['methods'].keys(), 'create')
    end
  end

  test "groups contents parameters" do
    get :index
    assert_response :success

    discovery_doc = JSON.parse(@response.body)

    group_index_params = discovery_doc['resources']['groups']['methods']['index']['parameters']
    group_contents_params = discovery_doc['resources']['groups']['methods']['contents']['parameters']

    assert_equal group_contents_params.keys.sort, (group_index_params.keys - ['select'] + ['uuid', 'recursive']).sort

    recursive_param = group_contents_params['recursive']
    assert_equal 'boolean', recursive_param['type']
    assert_equal false, recursive_param['required']
    assert_equal 'query', recursive_param['location']
  end

  test "collections index parameters" do
    get :index
    assert_response :success

    discovery_doc = JSON.parse(@response.body)

    specimens_index_params = discovery_doc['resources']['specimens']['methods']['index']['parameters']  # no changes from super
    coll_index_params = discovery_doc['resources']['collections']['methods']['index']['parameters']

    assert_equal coll_index_params.keys.sort, (specimens_index_params.keys + ['include_trash']).sort

    include_trash_param = coll_index_params['include_trash']
    assert_equal 'boolean', include_trash_param['type']
    assert_equal false, include_trash_param['required']
    assert_equal 'query', include_trash_param['location']
  end
end
