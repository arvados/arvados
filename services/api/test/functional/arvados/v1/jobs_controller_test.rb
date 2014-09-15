require 'test_helper'
require 'helpers/git_test_helper'

class Arvados::V1::JobsControllerTest < ActionController::TestCase

  include GitTestHelper

  test "submit a job" do
    authorize_with :active
    post :create, job: {
      script: "hash",
      script_version: "master",
      repository: "foo",
      script_parameters: {}
    }
    assert_response :success
    assert_not_nil assigns(:object)
    new_job = JSON.parse(@response.body)
    assert_not_nil new_job['uuid']
    assert_not_nil new_job['script_version'].match(/^[0-9a-f]{40}$/)
    assert_equal 0, new_job['priority']
  end

  test "normalize output and log uuids when creating job" do
    authorize_with :active
    post :create, job: {
      script: "hash",
      script_version: "master",
      script_parameters: {},
      repository: "foo",
      started_at: Time.now,
      finished_at: Time.now,
      running: false,
      success: true,
      output: 'd41d8cd98f00b204e9800998ecf8427e+0+K@xyzzy',
      log: 'd41d8cd98f00b204e9800998ecf8427e+0+K@xyzzy'
    }
    assert_response :success
    assert_not_nil assigns(:object)
    new_job = JSON.parse(@response.body)
    assert_equal 'd41d8cd98f00b204e9800998ecf8427e+0', new_job['log']
    assert_equal 'd41d8cd98f00b204e9800998ecf8427e+0', new_job['output']
    version = new_job['script_version']

    # Make sure version doesn't get mangled by normalize
    assert_not_nil version.match(/^[0-9a-f]{40}$/)
    put :update, {
      id: new_job['uuid'],
      job: {
        log: new_job['log']
      }
    }
    assert_equal version, JSON.parse(@response.body)['script_version']
  end

  test "cancel a running job" do
    # We need to verify that "cancel" creates a trigger file, so first
    # let's make sure there is no stale trigger file.
    begin
      File.unlink(Rails.configuration.crunch_refresh_trigger)
    rescue Errno::ENOENT
    end

    authorize_with :active
    put :update, {
      id: jobs(:running).uuid,
      job: {
        cancelled_at: 4.day.ago
      }
    }
    assert_response :success
    assert_not_nil assigns(:object)
    job = JSON.parse(@response.body)
    assert_not_nil job['uuid']
    assert_not_nil job['cancelled_at']
    assert_not_nil job['cancelled_by_user_uuid']
    assert_not_nil job['cancelled_by_client_uuid']
    assert_equal(true, Time.parse(job['cancelled_at']) > 1.minute.ago,
                 'server should correct bogus cancelled_at ' +
                 job['cancelled_at'])
    assert_equal(true,
                 File.exists?(Rails.configuration.crunch_refresh_trigger),
                 'trigger file should be created when job is cancelled')

    put :update, {
      id: jobs(:running).uuid,
      job: {
        cancelled_at: nil
      }
    }
    job = JSON.parse(@response.body)
    assert_not_nil job['cancelled_at'], 'un-cancelled job stays cancelled'
  end

  test "update a job without failing script_version check" do
    authorize_with :admin
    put :update, {
      id: jobs(:uses_nonexistent_script_version).uuid,
      job: {
        owner_uuid: users(:admin).uuid
      }
    }
    assert_response :success
    put :update, {
      id: jobs(:uses_nonexistent_script_version).uuid,
      job: {
        owner_uuid: users(:active).uuid
      }
    }
    assert_response :success
  end

  test "search jobs by uuid with >= query" do
    authorize_with :active
    get :index, {
      filters: [['uuid', '>=', 'zzzzz-8i9sb-pshmckwoma9plh7']]
    }
    assert_response :success
    found = assigns(:objects).collect(&:uuid)
    assert_equal true, !!found.index('zzzzz-8i9sb-pshmckwoma9plh7')
    assert_equal false, !!found.index('zzzzz-8i9sb-4cf0nhn6xte809j')
  end

  test "search jobs by uuid with <= query" do
    authorize_with :active
    get :index, {
      filters: [['uuid', '<=', 'zzzzz-8i9sb-pshmckwoma9plh7']]
    }
    assert_response :success
    found = assigns(:objects).collect(&:uuid)
    assert_equal true, !!found.index('zzzzz-8i9sb-pshmckwoma9plh7')
    assert_equal true, !!found.index('zzzzz-8i9sb-4cf0nhn6xte809j')
  end

  test "search jobs by uuid with >= and <= query" do
    authorize_with :active
    get :index, {
      filters: [['uuid', '>=', 'zzzzz-8i9sb-pshmckwoma9plh7'],
              ['uuid', '<=', 'zzzzz-8i9sb-pshmckwoma9plh7']]
    }
    assert_response :success
    found = assigns(:objects).collect(&:uuid)
    assert_equal found, ['zzzzz-8i9sb-pshmckwoma9plh7']
  end

  test "search jobs by uuid with < query" do
    authorize_with :active
    get :index, {
      filters: [['uuid', '<', 'zzzzz-8i9sb-pshmckwoma9plh7']]
    }
    assert_response :success
    found = assigns(:objects).collect(&:uuid)
    assert_equal false, !!found.index('zzzzz-8i9sb-pshmckwoma9plh7')
    assert_equal true, !!found.index('zzzzz-8i9sb-4cf0nhn6xte809j')
  end

  test "search jobs by uuid with like query" do
    authorize_with :active
    get :index, {
      filters: [['uuid', 'like', '%hmckwoma9pl%']]
    }
    assert_response :success
    found = assigns(:objects).collect(&:uuid)
    assert_equal found, ['zzzzz-8i9sb-pshmckwoma9plh7']
  end

  test "search jobs by uuid with 'in' query" do
    authorize_with :active
    get :index, {
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
    get :index, {
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
        get :index, {
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
    get :index, {
      filters: [['started_at', '<', Time.now.to_s]]
    }
    assert_response :success
    found = assigns(:objects).collect(&:uuid)
    assert_equal true, !!found.index('zzzzz-8i9sb-pshmckwoma9plh7')
  end

  test "search jobs by started_at with > query" do
    authorize_with :active
    get :index, {
      filters: [['started_at', '>', Time.now.to_s]]
    }
    assert_response :success
    assert_equal 0, assigns(:objects).count
  end

  test "search jobs by started_at with >= query on metric date" do
    authorize_with :active
    get :index, {
      filters: [['started_at', '>=', '2014-01-01']]
    }
    assert_response :success
    found = assigns(:objects).collect(&:uuid)
    assert_equal true, !!found.index('zzzzz-8i9sb-pshmckwoma9plh7')
  end

  test "search jobs by started_at with >= query on metric date and time" do
    authorize_with :active
    get :index, {
      filters: [['started_at', '>=', '2014-01-01 01:23:45']]
    }
    assert_response :success
    found = assigns(:objects).collect(&:uuid)
    assert_equal true, !!found.index('zzzzz-8i9sb-pshmckwoma9plh7')
  end

  test "search jobs with 'any' operator" do
    authorize_with :active
    get :index, {
      where: { any: ['contains', 'pshmckw'] }
    }
    assert_response :success
    found = assigns(:objects).collect(&:uuid)
    assert_equal 0, found.index('zzzzz-8i9sb-pshmckwoma9plh7')
    assert_equal 1, found.count
  end

  test "search jobs by nonexistent column with < query" do
    authorize_with :active
    get :index, {
      filters: [['is_borked', '<', 'fizzbuzz']]
    }
    assert_response 422
  end

  test "finish a job" do
    authorize_with :active
    put :update, {
      id: jobs(:nearly_finished_job).uuid,
      job: {
        output: '551392cc37a317abf865b95f66f4ef94+101',
        log: '9215de2a951a721f5f156bc08cf63ad7+93',
        tasks_summary: {done: 1, running: 0, todo: 0, failed: 0},
        success: true,
        running: false,
        finished_at: Time.now.to_s
      }
    }
    assert_response :success
  end

  [:active, :admin].each do |which_token|
    test "get job queue as #{which_token} user" do
      authorize_with which_token
      get :queue
      assert_response :success
      assert_operator 1, :<=, assigns(:objects).count
    end
    test "get job queue as #{which_token} user, with a filter" do
      authorize_with which_token
      get :queue, { filters: [['script','=','foo']] }
      assert_response :success
      assert_equal ['foo'], assigns(:objects).collect(&:script).uniq
    end
  end

end
