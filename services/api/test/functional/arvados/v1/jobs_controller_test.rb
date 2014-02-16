require 'test_helper'

class Arvados::V1::JobsControllerTest < ActionController::TestCase

  test "submit a job" do
    authorize_with :active
    post :create, job: {
      script: "hash",
      script_version: "master",
      script_parameters: {}
    }
    assert_response :success
    assert_not_nil assigns(:object)
    new_job = JSON.parse(@response.body)
    assert_not_nil new_job['uuid']
    assert_not_nil new_job['script_version'].match(/^[0-9a-f]{40}$/)
  end

  test "normalize output and log uuids when creating job" do
    authorize_with :active
    post :create, job: {
      script: "hash",
      script_version: "master",
      script_parameters: {},
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
end
