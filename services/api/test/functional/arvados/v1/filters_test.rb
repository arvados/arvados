# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class Arvados::V1::FiltersTest < ActionController::TestCase
  test '"not in" filter passes null values' do
    @controller = Arvados::V1::GroupsController.new
    authorize_with :admin
    get :index, {
      filters: [ ['group_class', 'not in', ['project']] ],
      controller: 'groups',
    }
    assert_response :success
    found = assigns(:objects)
    assert_includes(found.collect(&:group_class), nil,
                    "'group_class not in ['project']' filter should pass null")
  end

  test 'error message for non-array element in filters array' do
    @controller = Arvados::V1::CollectionsController.new
    authorize_with :active
    get :index, {
      filters: [{bogus: 'filter'}],
    }
    assert_response 422
    assert_match(/Invalid element in filters array/,
                 json_response['errors'].join(' '))
  end

  test 'error message for full text search on a specific column' do
    @controller = Arvados::V1::CollectionsController.new
    authorize_with :active
    get :index, {
      filters: [['uuid', '@@', 'abcdef']],
    }
    assert_response 422
    assert_match(/not supported/, json_response['errors'].join(' '))
  end

  test 'difficult characters in full text search' do
    @controller = Arvados::V1::CollectionsController.new
    authorize_with :active
    get :index, {
      filters: [['any', '@@', 'a|b"c']],
    }
    assert_response :success
    # (Doesn't matter so much which results are returned.)
  end

  test 'array operand in full text search' do
    @controller = Arvados::V1::CollectionsController.new
    authorize_with :active
    get :index, {
      filters: [['any', '@@', ['abc', 'def']]],
    }
    assert_response 422
    assert_match(/not supported/, json_response['errors'].join(' '))
  end

  test 'api responses provide timestamps with nanoseconds' do
    @controller = Arvados::V1::CollectionsController.new
    authorize_with :active
    get :index
    assert_response :success
    assert_not_empty json_response['items']
    json_response['items'].each do |item|
      %w(created_at modified_at).each do |attr|
        # Pass fixtures with null timestamps.
        next if item[attr].nil?
        assert_match(/^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d.\d{9}Z$/, item[attr])
      end
    end
  end

  %w(< > <= >= =).each do |operator|
    test "timestamp #{operator} filters work with nanosecond precision" do
      # Python clients like Node Manager rely on this exact format.
      # If you must change this format for some reason, make sure you
      # coordinate the change with them.
      expect_match = !!operator.index('=')
      mine = act_as_user users(:active) do
        Collection.create!(manifest_text: '')
      end
      timestamp = mine.modified_at.strftime('%Y-%m-%dT%H:%M:%S.%NZ')
      @controller = Arvados::V1::CollectionsController.new
      authorize_with :active
      get :index, {
        filters: [['modified_at', operator, timestamp],
                  ['uuid', '=', mine.uuid]],
      }
      assert_response :success
      uuids = json_response['items'].map { |item| item['uuid'] }
      if expect_match
        assert_includes uuids, mine.uuid
      else
        assert_not_includes uuids, mine.uuid
      end
    end
  end

  test "full text search with count='none'" do
    @controller = Arvados::V1::GroupsController.new
    authorize_with :admin

    get :contents, {
      format: :json,
      count: 'none',
      limit: 1000,
      filters: [['any', '@@', Rails.configuration.uuid_prefix]],
    }

    assert_response :success

    all_objects = Hash.new(0)
    json_response['items'].map{|o| o['kind']}.each{|t| all_objects[t] += 1}

    assert_equal true, all_objects['arvados#group']>0
    assert_equal true, all_objects['arvados#job']>0
    assert_equal true, all_objects['arvados#pipelineInstance']>0
    assert_equal true, all_objects['arvados#pipelineTemplate']>0

    # Perform test again mimicking a second page request with:
    # last_object_class = PipelineInstance
    #   and hence groups and jobs should not be included in the response
    # offset = 5, which means first 5 pipeline instances were already received in page 1
    #   and hence the remaining pipeline instances and all other object types should be included in the response

    @test_counter = 0  # Reset executed action counter

    @controller = Arvados::V1::GroupsController.new

    get :contents, {
      format: :json,
      count: 'none',
      limit: 1000,
      offset: '5',
      last_object_class: 'PipelineInstance',
      filters: [['any', '@@', Rails.configuration.uuid_prefix]],
    }

    assert_response :success

    second_page = Hash.new(0)
    json_response['items'].map{|o| o['kind']}.each{|t| second_page[t] += 1}

    assert_equal false, second_page.include?('arvados#group')
    assert_equal false, second_page.include?('arvados#job')
    assert_equal true, second_page['arvados#pipelineInstance']>0
    assert_equal all_objects['arvados#pipelineInstance'], second_page['arvados#pipelineInstance']+5
    assert_equal true, second_page['arvados#pipelineTemplate']>0
  end
end
