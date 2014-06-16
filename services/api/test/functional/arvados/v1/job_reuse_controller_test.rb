require 'test_helper'
load 'test/functional/arvados/v1/git_setup.rb'

class Arvados::V1::JobReuseControllerTest < ActionController::TestCase
  fixtures :repositories, :users, :jobs, :links, :collections

  # See git_setup.rb for the commit log for test.git.tar
  include GitSetup

  setup do
    @controller = Arvados::V1::JobsController.new
    authorize_with :active
  end

  test "reuse job with no_reuse=false" do
    post :create, job: {
      no_reuse: false,
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

  test "reuse job with find_or_create=true" do
    post :create, {
      job: {
        script: "hash",
        script_version: "4fe459abe02d9b365932b8f5dc419439ab4e2577",
        repository: "foo",
        script_parameters: {
          input: 'fa7aeb5140e2848d39b416daeef4ffc5+45',
          an_integer: '1'
        }
      },
      find_or_create: true
    }
    assert_response :success
    assert_not_nil assigns(:object)
    new_job = JSON.parse(@response.body)
    assert_equal 'zzzzz-8i9sb-cjs4pklxxjykqqq', new_job['uuid']
    assert_equal '4fe459abe02d9b365932b8f5dc419439ab4e2577', new_job['script_version']
  end

  test "do not reuse job because no_reuse=true" do
    post :create, {
      job: {
        no_reuse: true,
        script: "hash",
        script_version: "4fe459abe02d9b365932b8f5dc419439ab4e2577",
        repository: "foo",
        script_parameters: {
          input: 'fa7aeb5140e2848d39b416daeef4ffc5+45',
          an_integer: '1'
        }
      }
    }
    assert_response :success
    assert_not_nil assigns(:object)
    new_job = JSON.parse(@response.body)
    assert_not_equal 'zzzzz-8i9sb-cjs4pklxxjykqqq', new_job['uuid']
    assert_equal '4fe459abe02d9b365932b8f5dc419439ab4e2577', new_job['script_version']
  end

  test "do not reuse job because find_or_create=false" do
    post :create, {
      job: {
        script: "hash",
        script_version: "4fe459abe02d9b365932b8f5dc419439ab4e2577",
        repository: "foo",
        script_parameters: {
          input: 'fa7aeb5140e2848d39b416daeef4ffc5+45',
          an_integer: '1'
        }
      },
      find_or_create: false
    }
    assert_response :success
    assert_not_nil assigns(:object)
    new_job = JSON.parse(@response.body)
    assert_not_equal 'zzzzz-8i9sb-cjs4pklxxjykqqq', new_job['uuid']
    assert_equal '4fe459abe02d9b365932b8f5dc419439ab4e2577', new_job['script_version']
  end

  test "test_cannot_reuse_job_no_output" do
    post :create, job: {
      no_reuse: false,
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
    post :create, job: {
      no_reuse: false,
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

  test "cannot_reuse_job_no_minimum_given_so_must_use_specified_commit" do
    post :create, job: {
      no_reuse: false,
      script: "hash",
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
    assert_not_equal 'zzzzz-8i9sb-cjs4pklxxjykqqq', new_job['uuid']
    assert_equal '077ba2ad3ea24a929091a9e6ce545c93199b8e57', new_job['script_version']
  end

  test "test_cannot_reuse_job_different_input" do
    post :create, job: {
      no_reuse: false,
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
    post :create, job: {
      no_reuse: false,
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

  test "test_can_reuse_job_submitted_nondeterministic" do
    post :create, job: {
      no_reuse: false,
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
    assert_equal 'zzzzz-8i9sb-cjs4pklxxjykqqq', new_job['uuid']
    assert_equal '4fe459abe02d9b365932b8f5dc419439ab4e2577', new_job['script_version']
  end

  test "test_cannot_reuse_job_past_nondeterministic" do
    post :create, job: {
      no_reuse: false,
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
    authorize_with :spectator
    post :create, job: {
      no_reuse: false,
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
    post :create, job: {
      no_reuse: false,
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

  test "cannot reuse job with find_or_create but excluded version" do
    post :create, {
      job: {
        script: "hash",
        script_version: "master",
        repository: "foo",
        script_parameters: {
          input: 'fa7aeb5140e2848d39b416daeef4ffc5+45',
          an_integer: '1'
        }
      },
      find_or_create: true,
      minimum_script_version: "31ce37fe365b3dc204300a3e4c396ad333ed0556",
      exclude_script_versions: ["tag1"],
    }
    assert_response :success
    assert_not_nil assigns(:object)
    new_job = JSON.parse(@response.body)
    assert_not_equal 'zzzzz-8i9sb-cjs4pklxxjykqqq', new_job['uuid']
    assert_equal '077ba2ad3ea24a929091a9e6ce545c93199b8e57', new_job['script_version']
  end

  test "can reuse a Job based on filters" do
    post(:create, {
           job: {
             script: "hash",
             script_version: "master",
             repository: "foo",
             script_parameters: {
               input: 'fa7aeb5140e2848d39b416daeef4ffc5+45',
               an_integer: '1'
             }
           },
           filters: [["script_version", "in range", "tag1"]],
           find_or_create: true,
         })
    assert_response :success
    assert_not_nil assigns(:object)
    new_job = JSON.parse(@response.body)
    assert_equal 'zzzzz-8i9sb-cjs4pklxxjykqqq', new_job['uuid']
    assert_equal '4fe459abe02d9b365932b8f5dc419439ab4e2577', new_job['script_version']
  end

  test "can not reuse a Job based on filters" do
    post(:create, {
           job: {
             script: "hash",
             script_version: "master",
             repository: "foo",
             script_parameters: {
               input: 'fa7aeb5140e2848d39b416daeef4ffc5+45',
               an_integer: '1'
             }
           },
           filters: [["script_version", "in range",
                      "31ce37fe365b3dc204300a3e4c396ad333ed0556"],
                     ["script_version", "not in", ["tag1"]]],
           find_or_create: true,
         })
    assert_response :success
    assert_not_nil assigns(:object)
    new_job = JSON.parse(@response.body)
    assert_not_equal 'zzzzz-8i9sb-cjs4pklxxjykqqq', new_job['uuid']
    assert_equal '077ba2ad3ea24a929091a9e6ce545c93199b8e57', new_job['script_version']
  end

  test "can not reuse a Job based on arbitrary filters" do
    post(:create, {
           job: {
             script: "hash",
             script_version: "4fe459abe02d9b365932b8f5dc419439ab4e2577",
             repository: "foo",
             script_parameters: {
               input: 'fa7aeb5140e2848d39b416daeef4ffc5+45',
               an_integer: '1'
             }
           },
           filters: [["created_at", "<", "2010-01-01T00:00:00Z"]],
           find_or_create: true,
         })
    assert_response :success
    assert_not_nil assigns(:object)
    new_job = JSON.parse(@response.body)
    assert_not_equal 'zzzzz-8i9sb-cjs4pklxxjykqqq', new_job['uuid']
    assert_equal '4fe459abe02d9b365932b8f5dc419439ab4e2577', new_job['script_version']
  end
end
