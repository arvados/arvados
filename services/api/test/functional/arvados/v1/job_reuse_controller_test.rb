require 'test_helper'
load 'test/functional/arvados/v1/git_setup.rb'

class Arvados::V1::JobReuseControllerTest < ActionController::TestCase
  fixtures :repositories, :users, :jobs, :links, :collections

  # See git_setup.rb for the commit log for test.git.tar
  include GitSetup

  test "test_reuse_job" do
    @controller = Arvados::V1::JobsController.new
    authorize_with :active
    post :create, job: {
      script: "hash",
      script_version: "4fe459abe02d9b365932b8f5dc419439ab4e2577",
      repository: "foo",
      script_parameters: {
        input: 'fa7aeb5140e2848d39b416daeef4ffc5+45',
        an_integer: '1'
      }
    }
    assert_response :success
    assert_not_nil assigns(:object)
    new_job = JSON.parse(@response.body)
    assert_equal 'zzzzz-8i9sb-cjs4pklxxjykqqq', new_job['uuid']
    assert_equal '4fe459abe02d9b365932b8f5dc419439ab4e2577', new_job['script_version']
  end

  test "test_cannot_reuse_job_no_output" do
    @controller = Arvados::V1::JobsController.new
    authorize_with :active
    post :create, job: {
      script: "hash",
      script_version: "4fe459abe02d9b365932b8f5dc419439ab4e2577",
      repository: "foo",
      script_parameters: {
        input: 'fa7aeb5140e2848d39b416daeef4ffc5+45',
        an_integer: '2'
      }
    }
    assert_response :success
    assert_not_nil assigns(:object)
    new_job = JSON.parse(@response.body)
    assert_not_equal 'zzzzz-8i9sb-cjs4pklxxjykppp', new_job['uuid']
  end

  test "test_reuse_job_range" do
    @controller = Arvados::V1::JobsController.new
    authorize_with :active
    post :create, job: {
      script: "hash",
      minimum_script_version: "tag1",
      script_version: "master",
      repository: "foo",
      script_parameters: {
        input: 'fa7aeb5140e2848d39b416daeef4ffc5+45',
        an_integer: '1'
      }
    }
    assert_response :success
    assert_not_nil assigns(:object)
    new_job = JSON.parse(@response.body)
    assert_equal 'zzzzz-8i9sb-cjs4pklxxjykqqq', new_job['uuid']
    assert_equal '4fe459abe02d9b365932b8f5dc419439ab4e2577', new_job['script_version']
  end

  test "test_cannot_reuse_job_different_input" do
    @controller = Arvados::V1::JobsController.new
    authorize_with :active
    post :create, job: {
      script: "hash",
      script_version: "4fe459abe02d9b365932b8f5dc419439ab4e2577",
      repository: "foo",
      script_parameters: {
        input: 'fa7aeb5140e2848d39b416daeef4ffc5+45',
        an_integer: '2'
      }
    }
    assert_response :success
    assert_not_nil assigns(:object)
    new_job = JSON.parse(@response.body)
    assert_not_equal 'zzzzz-8i9sb-cjs4pklxxjykqqq', new_job['uuid']
    assert_equal '4fe459abe02d9b365932b8f5dc419439ab4e2577', new_job['script_version']
  end

  test "test_cannot_reuse_job_different_version" do
    @controller = Arvados::V1::JobsController.new
    authorize_with :active
    post :create, job: {
      script: "hash",
      script_version: "master",
      repository: "foo",
      script_parameters: {
        input: 'fa7aeb5140e2848d39b416daeef4ffc5+45',
        an_integer: '2'
      }
    }
    assert_response :success
    assert_not_nil assigns(:object)
    new_job = JSON.parse(@response.body)
    assert_not_equal 'zzzzz-8i9sb-cjs4pklxxjykqqq', new_job['uuid']
    assert_equal '077ba2ad3ea24a929091a9e6ce545c93199b8e57', new_job['script_version']
  end

  test "test_cannot_reuse_job_submitted_nondeterministic" do
    @controller = Arvados::V1::JobsController.new
    authorize_with :active
    post :create, job: {
      script: "hash",
      script_version: "4fe459abe02d9b365932b8f5dc419439ab4e2577",
      repository: "foo",
      script_parameters: {
        input: 'fa7aeb5140e2848d39b416daeef4ffc5+45',
        an_integer: '1'
      },
      nondeterministic: true
    }
    assert_response :success
    assert_not_nil assigns(:object)
    new_job = JSON.parse(@response.body)
    assert_not_equal 'zzzzz-8i9sb-cjs4pklxxjykqqq', new_job['uuid']
    assert_equal '4fe459abe02d9b365932b8f5dc419439ab4e2577', new_job['script_version']
  end

  test "test_cannot_reuse_job_past_nondeterministic" do
    @controller = Arvados::V1::JobsController.new
    authorize_with :active
    post :create, job: {
      script: "hash2",
      script_version: "4fe459abe02d9b365932b8f5dc419439ab4e2577",
      repository: "foo",
      script_parameters: {
        input: 'fa7aeb5140e2848d39b416daeef4ffc5+45',
        an_integer: '1'
      }
    }
    assert_response :success
    assert_not_nil assigns(:object)
    new_job = JSON.parse(@response.body)
    assert_not_equal 'zzzzz-8i9sb-cjs4pklxxjykyyy', new_job['uuid']
    assert_equal '4fe459abe02d9b365932b8f5dc419439ab4e2577', new_job['script_version']
  end

  test "test_cannot_reuse_job_no_permission" do
    @controller = Arvados::V1::JobsController.new
    authorize_with :spectator
    post :create, job: {
      script: "hash",
      script_version: "4fe459abe02d9b365932b8f5dc419439ab4e2577",
      repository: "foo",
      script_parameters: {
        input: 'fa7aeb5140e2848d39b416daeef4ffc5+45',
        an_integer: '1'
      }
    }
    assert_response :success
    assert_not_nil assigns(:object)
    new_job = JSON.parse(@response.body)
    assert_not_equal 'zzzzz-8i9sb-cjs4pklxxjykqqq', new_job['uuid']
    assert_equal '4fe459abe02d9b365932b8f5dc419439ab4e2577', new_job['script_version']
  end

  test "test_cannot_reuse_job_excluded" do
    @controller = Arvados::V1::JobsController.new
    authorize_with :active
    post :create, job: {
      script: "hash",
      minimum_script_version: "31ce37fe365b3dc204300a3e4c396ad333ed0556",
      script_version: "master",
      repository: "foo",
      exclude_script_versions: ["tag1"],
      script_parameters: {
        input: 'fa7aeb5140e2848d39b416daeef4ffc5+45',
        an_integer: '1'
      }
    }
    assert_response :success
    assert_not_nil assigns(:object)
    new_job = JSON.parse(@response.body)
    assert_not_equal 'zzzzz-8i9sb-cjs4pklxxjykqqq', new_job['uuid']
    assert_equal '077ba2ad3ea24a929091a9e6ce545c93199b8e57', new_job['script_version']
  end


end
