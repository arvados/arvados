# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'
require 'helpers/git_test_helper'

class Arvados::V1::JobsControllerTest < ActionController::TestCase

  test "search jobs by uuid with >= query" do
    authorize_with :active
    get :index, params: {
      filters: [['uuid', '>=', 'zzzzz-8i9sb-pshmckwoma9plh7']]
    }
    assert_response :success
    found = assigns(:objects).collect(&:uuid)
    assert_equal true, !!found.index('zzzzz-8i9sb-pshmckwoma9plh7')
    assert_equal false, !!found.index('zzzzz-8i9sb-4cf0nhn6xte809j')
  end

  test "search jobs by uuid with <= query" do
    authorize_with :active
    get :index, params: {
      filters: [['uuid', '<=', 'zzzzz-8i9sb-pshmckwoma9plh7']]
    }
    assert_response :success
    found = assigns(:objects).collect(&:uuid)
    assert_equal true, !!found.index('zzzzz-8i9sb-pshmckwoma9plh7')
    assert_equal true, !!found.index('zzzzz-8i9sb-4cf0nhn6xte809j')
  end

  test "search jobs by uuid with >= and <= query" do
    authorize_with :active
    get :index, params: {
      filters: [['uuid', '>=', 'zzzzz-8i9sb-pshmckwoma9plh7'],
              ['uuid', '<=', 'zzzzz-8i9sb-pshmckwoma9plh7']]
    }
    assert_response :success
    found = assigns(:objects).collect(&:uuid)
    assert_equal found, ['zzzzz-8i9sb-pshmckwoma9plh7']
  end

  test "search jobs by uuid with < query" do
    authorize_with :active
    get :index, params: {
      filters: [['uuid', '<', 'zzzzz-8i9sb-pshmckwoma9plh7']]
    }
    assert_response :success
    found = assigns(:objects).collect(&:uuid)
    assert_equal false, !!found.index('zzzzz-8i9sb-pshmckwoma9plh7')
    assert_equal true, !!found.index('zzzzz-8i9sb-4cf0nhn6xte809j')
  end

  test "search jobs by uuid with like query" do
    authorize_with :active
    get :index, params: {
      filters: [['uuid', 'like', '%hmckwoma9pl%']]
    }
    assert_response :success
    found = assigns(:objects).collect(&:uuid)
    assert_equal found, ['zzzzz-8i9sb-pshmckwoma9plh7']
  end

  test "search jobs by uuid with 'in' query" do
    authorize_with :active
    get :index, params: {
      filters: [['uuid', 'in', ['zzzzz-8i9sb-4cf0nhn6xte809j',
                                'zzzzz-8i9sb-pshmckwoma9plh7']]]
    }
    assert_response :success
    found = assigns(:objects).collect(&:uuid)
    assert_equal found.sort, ['zzzzz-8i9sb-4cf0nhn6xte809j',
                              'zzzzz-8i9sb-pshmckwoma9plh7']
  end

  test "search jobs by uuid with 'not in' query" do
    exclude_uuids = [jobs(:running).uuid,
                     jobs(:running_cancelled).uuid]
    authorize_with :active
    get :index, params: {
      filters: [['uuid', 'not in', exclude_uuids]]
    }
    assert_response :success
    found = assigns(:objects).collect(&:uuid)
    assert_not_empty found, "'not in' query returned nothing"
    assert_empty(found & exclude_uuids,
                 "'not in' query returned uuids I asked not to get")
  end

  ['=', '!='].each do |operator|
    [['uuid', 'zzzzz-8i9sb-pshmckwoma9plh7'],
     ['output', nil]].each do |attr, operand|
      test "search jobs with #{attr} #{operator} #{operand.inspect} query" do
        authorize_with :active
        get :index, params: {
          filters: [[attr, operator, operand]]
        }
        assert_response :success
        values = assigns(:objects).collect { |x| x.send(attr) }
        assert_not_empty values, "query should return non-empty result"
        if operator == '='
          assert_empty values - [operand], "query results do not satisfy query"
        else
          assert_empty values & [operand], "query results do not satisfy query"
        end
      end
    end
  end

  test "search jobs by started_at with < query" do
    authorize_with :active
    get :index, params: {
      filters: [['started_at', '<', Time.now.to_s]]
    }
    assert_response :success
    found = assigns(:objects).collect(&:uuid)
    assert_equal true, !!found.index('zzzzz-8i9sb-pshmckwoma9plh7')
  end

  test "search jobs by started_at with > query" do
    authorize_with :active
    get :index, params: {
      filters: [['started_at', '>', Time.now.to_s]]
    }
    assert_response :success
    assert_equal 0, assigns(:objects).count
  end

  test "search jobs by started_at with >= query on metric date" do
    authorize_with :active
    get :index, params: {
      filters: [['started_at', '>=', '2014-01-01']]
    }
    assert_response :success
    found = assigns(:objects).collect(&:uuid)
    assert_equal true, !!found.index('zzzzz-8i9sb-pshmckwoma9plh7')
  end

  test "search jobs by started_at with >= query on metric date and time" do
    authorize_with :active
    get :index, params: {
      filters: [['started_at', '>=', '2014-01-01 01:23:45']]
    }
    assert_response :success
    found = assigns(:objects).collect(&:uuid)
    assert_equal true, !!found.index('zzzzz-8i9sb-pshmckwoma9plh7')
  end

  test "search jobs with 'any' operator" do
    authorize_with :active
    get :index, params: {
      where: { any: ['contains', 'pshmckw'] }
    }
    assert_response :success
    found = assigns(:objects).collect(&:uuid)
    assert_equal 0, found.index('zzzzz-8i9sb-pshmckwoma9plh7')
    assert_equal 1, found.count
  end

  test "search jobs by nonexistent column with < query" do
    authorize_with :active
    get :index, params: {
      filters: [['is_borked', '<', 'fizzbuzz']]
    }
    assert_response 422
  end

  [:spectator, :admin].each_with_index do |which_token, i|
    test "get job queue as #{which_token} user" do
      authorize_with which_token
      get :queue
      assert_response :success
      assert_equal 0, assigns(:objects).count
    end
  end

  test "job includes assigned nodes" do
    authorize_with :active
    get :show, params: {id: jobs(:nearly_finished_job).uuid}
    assert_response :success
    assert_equal([nodes(:busy).uuid], json_response["node_uuids"])
  end

  test 'get job with components' do
    authorize_with :active
    get :show, params: {id: jobs(:running_job_with_components).uuid}
    assert_response :success
    assert_not_nil json_response["components"]
    assert_equal ["component1", "component2"], json_response["components"].keys
  end
end
