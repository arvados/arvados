# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class Arvados::V1::FiltersTest < ActionController::TestCase
  test '"not in" filter passes null values' do
    @controller = Arvados::V1::ContainerRequestsController.new
    authorize_with :admin
    get :index, params: {
      filters: [ ['container_uuid', 'not in', ['zzzzz-dz642-queuedcontainer', 'zzzzz-dz642-runningcontainr']] ],
      controller: 'container_requests',
    }
    assert_response :success
    found = assigns(:objects)
    assert_includes(found.collect(&:container_uuid), nil,
                    "'container_uuid not in [zzzzz-dz642-queuedcontainer, zzzzz-dz642-runningcontainr]' filter should pass null")
  end

  test 'error message for non-array element in filters array' do
    @controller = Arvados::V1::CollectionsController.new
    authorize_with :active
    get :index, params: {
      filters: [{bogus: 'filter'}],
    }
    assert_response 422
    assert_match(/Invalid element in filters array/,
                 json_response['errors'].join(' '))
  end

  test 'error message for full text search on a specific column' do
    @controller = Arvados::V1::CollectionsController.new
    authorize_with :active
    get :index, params: {
      filters: [['uuid', '@@', 'abcdef']],
    }
    assert_response 422
    assert_match(/not supported/, json_response['errors'].join(' '))
  end

  test 'difficult characters in full text search' do
    @controller = Arvados::V1::CollectionsController.new
    authorize_with :active
    get :index, params: {
      filters: [['any', '@@', 'a|b"c']],
    }
    assert_response :success
    # (Doesn't matter so much which results are returned.)
  end

  test 'array operand in full text search' do
    @controller = Arvados::V1::CollectionsController.new
    authorize_with :active
    get :index, params: {
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
      get :index, params: {
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

    get :contents, params: {
      format: :json,
      count: 'none',
      limit: 1000,
      filters: [['any', '@@', Rails.configuration.ClusterID]],
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

    get :contents, params: {
      format: :json,
      count: 'none',
      limit: 1000,
      offset: '5',
      last_object_class: 'PipelineInstance',
      filters: [['any', '@@', Rails.configuration.ClusterID]],
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

  [['prop1', '=', 'value1', [:collection_with_prop1_value1], [:collection_with_prop1_value2, :collection_with_prop2_1]],
   ['prop1', '!=', 'value1', [:collection_with_prop1_value2, :collection_with_prop2_1], [:collection_with_prop1_value1]],
   ['prop1', 'exists', true, [:collection_with_prop1_value1, :collection_with_prop1_value2, :collection_with_prop1_value3, :collection_with_prop1_other1], [:collection_with_prop2_1]],
   ['prop1', 'exists', false, [:collection_with_prop2_1], [:collection_with_prop1_value1, :collection_with_prop1_value2, :collection_with_prop1_value3, :collection_with_prop1_other1]],
   ['prop1', 'in', ['value1', 'value2'], [:collection_with_prop1_value1, :collection_with_prop1_value2], [:collection_with_prop1_value3, :collection_with_prop2_1]],
   ['prop1', 'in', ['value1', 'valueX'], [:collection_with_prop1_value1], [:collection_with_prop1_value3, :collection_with_prop2_1]],
   ['prop1', 'not in', ['value1', 'value2'], [:collection_with_prop1_value3, :collection_with_prop1_other1, :collection_with_prop2_1], [:collection_with_prop1_value1, :collection_with_prop1_value2]],
   ['prop1', 'not in', ['value1', 'valueX'], [:collection_with_prop1_value2, :collection_with_prop1_value3, :collection_with_prop1_other1, :collection_with_prop2_1], [:collection_with_prop1_value1]],
   ['prop1', '>', 'value2', [:collection_with_prop1_value3], [:collection_with_prop1_other1, :collection_with_prop1_value1]],
   ['prop1', '<', 'value2', [:collection_with_prop1_other1, :collection_with_prop1_value1], [:collection_with_prop1_value2, :collection_with_prop1_value2]],
   ['prop1', '<=', 'value2', [:collection_with_prop1_other1, :collection_with_prop1_value1, :collection_with_prop1_value2], [:collection_with_prop1_value3]],
   ['prop1', '>=', 'value2', [:collection_with_prop1_value2, :collection_with_prop1_value3], [:collection_with_prop1_other1, :collection_with_prop1_value1]],
   ['prop1', 'like', 'value%', [:collection_with_prop1_value1, :collection_with_prop1_value2, :collection_with_prop1_value3], [:collection_with_prop1_other1]],
   ['prop1', 'like', '%1', [:collection_with_prop1_value1, :collection_with_prop1_other1], [:collection_with_prop1_value2, :collection_with_prop1_value3]],
   ['prop1', 'ilike', 'VALUE%', [:collection_with_prop1_value1, :collection_with_prop1_value2, :collection_with_prop1_value3], [:collection_with_prop1_other1]],
   ['prop2', '>',  1, [:collection_with_prop2_5], [:collection_with_prop2_1]],
   ['prop2', '<',  5, [:collection_with_prop2_1], [:collection_with_prop2_5]],
   ['prop2', '<=', 5, [:collection_with_prop2_1, :collection_with_prop2_5], []],
   ['prop2', '>=', 1, [:collection_with_prop2_1, :collection_with_prop2_5], []],
   ['<http://schema.org/example>', '=', "value1", [:collection_with_uri_prop], []],
   ['listprop', 'contains', 'elem1', [:collection_with_list_prop_odd, :collection_with_listprop_elem1], [:collection_with_list_prop_even]],
   ['listprop', '=', 'elem1', [:collection_with_listprop_elem1], [:collection_with_list_prop_odd]],
   ['listprop', 'contains', 5, [:collection_with_list_prop_odd], [:collection_with_list_prop_even, :collection_with_listprop_elem1]],
   ['listprop', 'contains', 'elem2', [:collection_with_list_prop_even], [:collection_with_list_prop_odd, :collection_with_listprop_elem1]],
   ['listprop', 'contains', 'ELEM2', [], [:collection_with_list_prop_even]],
   ['listprop', 'contains', 'elem8', [], [:collection_with_list_prop_even]],
   ['listprop', 'contains', 4, [:collection_with_list_prop_even], [:collection_with_list_prop_odd, :collection_with_listprop_elem1]],
  ].each do |prop, op, opr, inc, ex|
    test "jsonb filter properties.#{prop} #{op} #{opr})" do
      @controller = Arvados::V1::CollectionsController.new
      authorize_with :admin
      get :index, params: {
            filters: SafeJSON.dump([ ["properties.#{prop}", op, opr] ]),
            limit: 1000
          }
      assert_response :success
      found = assigns(:objects).collect(&:uuid)

      inc.each do |i|
        assert_includes(found, collections(i).uuid)
      end

      ex.each do |e|
        assert_not_includes(found, collections(e).uuid)
      end
    end
  end

  test "jsonb hash 'exists' and '!=' filter" do
    @controller = Arvados::V1::CollectionsController.new
    authorize_with :admin
    get :index, params: {
      filters: [ ['properties.prop1', 'exists', true], ['properties.prop1', '!=', 'value1'] ]
    }
    assert_response :success
    found = assigns(:objects).collect(&:uuid)
    assert_equal found.length, 3
    assert_not_includes(found, collections(:collection_with_prop1_value1).uuid)
    assert_includes(found, collections(:collection_with_prop1_value2).uuid)
    assert_includes(found, collections(:collection_with_prop1_value3).uuid)
    assert_includes(found, collections(:collection_with_prop1_other1).uuid)
  end

  test "jsonb array 'exists'" do
    @controller = Arvados::V1::CollectionsController.new
    authorize_with :admin
    get :index, params: {
      filters: [ ['storage_classes_confirmed.default', 'exists', true] ]
    }
    assert_response :success
    found = assigns(:objects).collect(&:uuid)
    assert_equal 2, found.length
    assert_not_includes(found,
      collections(:storage_classes_desired_default_unconfirmed).uuid)
    assert_includes(found,
      collections(:storage_classes_desired_default_confirmed_default).uuid)
    assert_includes(found,
      collections(:storage_classes_desired_archive_confirmed_default).uuid)
  end

  test "jsonb hash alternate form 'exists' and '!=' filter" do
    @controller = Arvados::V1::CollectionsController.new
    authorize_with :admin
    get :index, params: {
      filters: [ ['properties', 'exists', 'prop1'], ['properties.prop1', '!=', 'value1'] ]
    }
    assert_response :success
    found = assigns(:objects).collect(&:uuid)
    assert_equal found.length, 3
    assert_not_includes(found, collections(:collection_with_prop1_value1).uuid)
    assert_includes(found, collections(:collection_with_prop1_value2).uuid)
    assert_includes(found, collections(:collection_with_prop1_value3).uuid)
    assert_includes(found, collections(:collection_with_prop1_other1).uuid)
  end

  test "jsonb array alternate form 'exists' filter" do
    @controller = Arvados::V1::CollectionsController.new
    authorize_with :admin
    get :index, params: {
      filters: [ ['storage_classes_confirmed', 'exists', 'default'] ]
    }
    assert_response :success
    found = assigns(:objects).collect(&:uuid)
    assert_equal 2, found.length
    assert_not_includes(found,
      collections(:storage_classes_desired_default_unconfirmed).uuid)
    assert_includes(found,
      collections(:storage_classes_desired_default_confirmed_default).uuid)
    assert_includes(found,
      collections(:storage_classes_desired_archive_confirmed_default).uuid)
  end

  test "jsonb 'exists' must be boolean" do
    @controller = Arvados::V1::CollectionsController.new
    authorize_with :admin
    get :index, params: {
      filters: [ ['properties.prop1', 'exists', nil] ]
    }
    assert_response 422
    assert_match(/Invalid operand '' for 'exists' must be true or false/,
                 json_response['errors'].join(' '))
  end

  test "jsonb checks column exists" do
    @controller = Arvados::V1::CollectionsController.new
    authorize_with :admin
    get :index, params: {
      filters: [ ['puppies.prop1', '=', 'value1'] ]
    }
    assert_response 422
    assert_match(/Invalid attribute 'puppies' for subproperty filter/,
                 json_response['errors'].join(' '))
  end

  test "jsonb checks column is valid" do
    @controller = Arvados::V1::CollectionsController.new
    authorize_with :admin
    get :index, params: {
      filters: [ ['name.prop1', '=', 'value1'] ]
    }
    assert_response 422
    assert_match(/Invalid attribute 'name' for subproperty filter/,
                 json_response['errors'].join(' '))
  end

  test "jsonb invalid operator" do
    @controller = Arvados::V1::CollectionsController.new
    authorize_with :admin
    get :index, params: {
      filters: [ ['properties.prop1', '###', 'value1'] ]
    }
    assert_response 422
    assert_match(/Invalid operator for subproperty search '###'/,
                 json_response['errors'].join(' '))
  end

  test "replication_desired = 2" do
    @controller = Arvados::V1::CollectionsController.new
    authorize_with :admin
    get :index, params: {
      filters: SafeJSON.dump([ ['replication_desired', '=', 2] ])
    }
    assert_response :success
    found = assigns(:objects).collect(&:uuid)
    assert_includes(found, collections(:replication_desired_2_unconfirmed).uuid)
    assert_includes(found, collections(:replication_desired_2_confirmed_2).uuid)
  end

  [
    [1, "foo"],
    [1, ["foo"]],
    [1, ["bar"]],
    [1, ["bar", "foo"]],
    [0, ["foo", "qux"]],
    [0, ["qux"]],
    [nil, []],
    [nil, [[]]],
    [nil, [["bogus"]]],
    [nil, [{"foo" => "bar"}]],
    [nil, {"foo" => "bar"}],
  ].each do |results, operand|
    test "storage_classes_desired contains #{operand.inspect}" do
      @controller = Arvados::V1::CollectionsController.new
      authorize_with(:active)
      c = Collection.create!(
        manifest_text: "",
        storage_classes_desired: ["foo", "bar", "baz"])
      get :index, params: {
            filters: [["storage_classes_desired", "contains", operand]],
          }
      if results.nil?
        assert_response 422
        next
      end
      assert_response :success
      assert_equal results, json_response["items"].length
      if results > 0
        assert_equal c.uuid, json_response["items"][0]["uuid"]
      end
    end
  end

  test "collections properties contains top level key" do
    @controller = Arvados::V1::CollectionsController.new
    authorize_with(:active)
    get :index, params: {
          filters: [["properties", "contains", "prop1"]],
        }
    assert_response :success
    assert_not_empty json_response["items"]
    json_response["items"].each do |c|
      assert c["properties"].has_key?("prop1")
    end
  end
end
