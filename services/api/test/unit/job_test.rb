# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'
require 'helpers/git_test_helper'
require 'helpers/docker_migration_helper'

class JobTest < ActiveSupport::TestCase
  include DockerMigrationHelper
  include GitTestHelper

  BAD_COLLECTION = "#{'f' * 32}+0"

  setup do
    set_user_from_auth :active
  end

  def job_attrs merge_me={}
    # Default (valid) set of attributes, with given overrides
    {
      script: "hash",
      script_version: "master",
      repository: "active/foo",
    }.merge(merge_me)
  end

  test "Job without Docker image doesn't get locator" do
    job = Job.new job_attrs
    assert job.valid?, job.errors.full_messages.to_s
    assert_nil job.docker_image_locator
  end

  { 'name' => [:links, :docker_image_collection_tag, :name],
    'hash' => [:links, :docker_image_collection_hash, :name],
    'locator' => [:collections, :docker_image, :portable_data_hash],
  }.each_pair do |spec_type, (fixture_type, fixture_name, fixture_attr)|
    test "Job initialized with Docker image #{spec_type} gets locator" do
      image_spec = send(fixture_type, fixture_name).send(fixture_attr)
      job = Job.new job_attrs(runtime_constraints:
                              {'docker_image' => image_spec})
      assert job.valid?, job.errors.full_messages.to_s
      assert_equal(collections(:docker_image).portable_data_hash, job.docker_image_locator)
    end

    test "Job modified with Docker image #{spec_type} gets locator" do
      job = Job.new job_attrs
      assert job.valid?, job.errors.full_messages.to_s
      assert_nil job.docker_image_locator
      image_spec = send(fixture_type, fixture_name).send(fixture_attr)
      job.runtime_constraints['docker_image'] = image_spec
      assert job.valid?, job.errors.full_messages.to_s
      assert_equal(collections(:docker_image).portable_data_hash, job.docker_image_locator)
    end
  end

  test "removing a Docker runtime constraint removes the locator" do
    image_locator = collections(:docker_image).portable_data_hash
    job = Job.new job_attrs(runtime_constraints:
                            {'docker_image' => image_locator})
    assert job.valid?, job.errors.full_messages.to_s
    assert_equal(image_locator, job.docker_image_locator)
    job.runtime_constraints = {}
    assert job.valid?, job.errors.full_messages.to_s + "after clearing runtime constraints"
    assert_nil job.docker_image_locator
  end

  test "locate a Docker image with a repository + tag" do
    image_repo, image_tag =
      links(:docker_image_collection_tag2).name.split(':', 2)
    job = Job.new job_attrs(runtime_constraints:
                            {'docker_image' => image_repo,
                              'docker_image_tag' => image_tag})
    assert job.valid?, job.errors.full_messages.to_s
    assert_equal(collections(:docker_image).portable_data_hash, job.docker_image_locator)
  end

  test "can't locate a Docker image with a nonexistent tag" do
    image_repo = links(:docker_image_collection_tag).name
    image_tag = '__nonexistent tag__'
    job = Job.new job_attrs(runtime_constraints:
                            {'docker_image' => image_repo,
                              'docker_image_tag' => image_tag})
    assert(job.invalid?, "Job with bad Docker tag valid")
  end

  [
    false,
    true
  ].each do |use_config|
    test "Job with no Docker image uses default docker image when configuration is set #{use_config}" do
      default_docker_image = collections(:docker_image)[:portable_data_hash]
      Rails.configuration.default_docker_image_for_jobs = default_docker_image if use_config

      job = Job.new job_attrs
      assert job.valid?, job.errors.full_messages.to_s

      if use_config
        refute_nil job.docker_image_locator
        assert_equal default_docker_image, job.docker_image_locator
      else
        assert_nil job.docker_image_locator
      end
    end
  end

  test "create a job with a disambiguated script_version branch name" do
    job = Job.
      new(script: "testscript",
          script_version: "heads/7387838c69a21827834586cc42b467ff6c63293b",
          repository: "active/shabranchnames",
          script_parameters: {})
    assert(job.save)
    assert_equal("abec49829bf1758413509b7ffcab32a771b71e81", job.script_version)
  end

  test "locate a Docker image with a partial hash" do
    image_hash = links(:docker_image_collection_hash).name[0..24]
    job = Job.new job_attrs(runtime_constraints:
                            {'docker_image' => image_hash})
    assert job.valid?, job.errors.full_messages.to_s + " with partial hash #{image_hash}"
    assert_equal(collections(:docker_image).portable_data_hash, job.docker_image_locator)
  end

  { 'name' => 'arvados_test_nonexistent',
    'hash' => 'f' * 64,
    'locator' => BAD_COLLECTION,
  }.each_pair do |spec_type, image_spec|
    test "Job validation fails with nonexistent Docker image #{spec_type}" do
      job = Job.new job_attrs(runtime_constraints:
                              {'docker_image' => image_spec})
      assert(job.invalid?, "nonexistent Docker image #{spec_type} was valid")
    end
  end

  test "Job validation fails with non-Docker Collection constraint" do
    job = Job.new job_attrs(runtime_constraints:
                            {'docker_image' => collections(:foo_file).uuid})
    assert(job.invalid?, "non-Docker Collection constraint was valid")
  end

  test "can create Job with Docker image Collection without Docker links" do
    image_uuid = collections(:unlinked_docker_image).portable_data_hash
    job = Job.new job_attrs(runtime_constraints: {"docker_image" => image_uuid})
    assert(job.valid?, "Job created with unlinked Docker image was invalid")
    assert_equal(image_uuid, job.docker_image_locator)
  end

  def check_attrs_unset(job, attrs)
    assert_empty(attrs.each_key.map { |key| job.send(key) }.compact,
                 "job has values for #{attrs.keys}")
  end

  def check_creation_prohibited(attrs)
    begin
      job = Job.new(job_attrs(attrs))
    rescue ActiveModel::MassAssignmentSecurity::Error
      # Test passes - expected attribute protection
    else
      check_attrs_unset(job, attrs)
    end
  end

  def check_modification_prohibited(attrs)
    job = Job.new(job_attrs)
    attrs.each_pair do |key, value|
      assert_raises(NoMethodError) { job.send("{key}=".to_sym, value) }
    end
    check_attrs_unset(job, attrs)
  end

  test "can't create Job with Docker image locator" do
    check_creation_prohibited(docker_image_locator: BAD_COLLECTION)
  end

  test "can't assign Docker image locator to Job" do
    check_modification_prohibited(docker_image_locator: BAD_COLLECTION)
  end

  [
   {script_parameters: ""},
   {script_parameters: []},
   {script_parameters: {["foo"] => ["bar"]}},
   {runtime_constraints: ""},
   {runtime_constraints: []},
   {tasks_summary: ""},
   {tasks_summary: []},
  ].each do |invalid_attrs|
    test "validation failures set error messages: #{invalid_attrs.to_json}" do
      # Ensure valid_attrs doesn't produce errors -- otherwise we will
      # not know whether errors reported below are actually caused by
      # invalid_attrs.
      Job.new(job_attrs).save!

      err = assert_raises(ArgumentError) do
        Job.new(job_attrs(invalid_attrs)).save!
      end
      assert_match /parameters|constraints|summary/, err.message
    end
  end

  test "invalid script_version" do
    invalid = {
      script_version: "no/branch/could/ever/possibly/have/this/name",
    }
    err = assert_raises(ActiveRecord::RecordInvalid) do
      Job.new(job_attrs(invalid)).save!
    end
    assert_match /Script version .* does not resolve to a commit/, err.message
  end

  [
    # Each test case is of the following format
    # Array of parameters where each parameter is of the format:
    #  attr name to be changed, attr value, and array of expectations (where each expectation is an array)
    [['running', false, [['state', 'Queued']]]],
    [['state', 'Running', [['started_at', 'not_nil']]]],
    [['is_locked_by_uuid', 'use_current_user_uuid', [['state', 'Queued']]], ['state', 'Running', [['running', true], ['started_at', 'not_nil'], ['success', 'nil']]]],
    [['running', false, [['state', 'Queued']]], ['state', 'Complete', [['success', true]]]],
    [['running', true, [['state', 'Running']]], ['cancelled_at', Time.now, [['state', 'Cancelled']]]],
    [['running', true, [['state', 'Running']]], ['state', 'Cancelled', [['cancelled_at', 'not_nil']]]],
    [['running', true, [['state', 'Running']]], ['success', true, [['state', 'Complete']]]],
    [['running', true, [['state', 'Running']]], ['success', false, [['state', 'Failed']]]],
    [['running', true, [['state', 'Running']]], ['state', 'Complete', [['success', true],['finished_at', 'not_nil']]]],
    [['running', true, [['state', 'Running']]], ['state', 'Failed', [['success', false],['finished_at', 'not_nil']]]],
    [['cancelled_at', Time.now, [['state', 'Cancelled']]], ['success', false, [['state', 'Cancelled'],['finished_at', 'nil'], ['cancelled_at', 'not_nil']]]],
    [['cancelled_at', Time.now, [['state', 'Cancelled'],['running', false]]], ['success', true, [['state', 'Cancelled'],['running', false],['finished_at', 'nil'],['cancelled_at', 'not_nil']]]],
    # potential migration cases
    [['state', nil, [['state', 'Queued']]]],
    [['state', nil, [['state', 'Queued']]], ['cancelled_at', Time.now, [['state', 'Cancelled']]]],
    [['running', true, [['state', 'Running']]], ['state', nil, [['state', 'Running']]]],
  ].each do |parameters|
    test "verify job status #{parameters}" do
      job = Job.create! job_attrs
      assert_equal 'Queued', job.state, "job.state"

      parameters.each do |parameter|
        expectations = parameter[2]
        if 'use_current_user_uuid' == parameter[1]
          parameter[1] = Thread.current[:user].uuid
        end

        if expectations.instance_of? Array
          job[parameter[0]] = parameter[1]
          assert_equal true, job.save, job.errors.full_messages.to_s
          expectations.each do |expectation|
            if expectation[1] == 'not_nil'
              assert_not_nil job[expectation[0]], expectation[0]
            elsif expectation[1] == 'nil'
              assert_nil job[expectation[0]], expectation[0]
            else
              assert_equal expectation[1], job[expectation[0]], expectation[0]
            end
          end
        else
          raise 'I do not know how to handle this expectation'
        end
      end
    end
  end

  test "Test job state changes" do
    all = ["Queued", "Running", "Complete", "Failed", "Cancelled"]
    valid = {"Queued" => all, "Running" => ["Complete", "Failed", "Cancelled"]}
    all.each do |start|
      all.each do |finish|
        if start != finish
          job = Job.create! job_attrs(state: start)
          assert_equal start, job.state
          job.state = finish
          job.save
          job.reload
          if valid[start] and valid[start].include? finish
            assert_equal finish, job.state
          else
            assert_equal start, job.state
          end
        end
      end
    end
  end

  test "Test job locking" do
    set_user_from_auth :active_trustedclient
    job = Job.create! job_attrs

    assert_equal "Queued", job.state

    # Should be able to lock successfully
    job.lock current_user.uuid
    assert_equal "Running", job.state

    assert_raises ArvadosModel::AlreadyLockedError do
      # Can't lock it again
      job.lock current_user.uuid
    end
    job.reload
    assert_equal "Running", job.state

    set_user_from_auth :project_viewer
    assert_raises ArvadosModel::AlreadyLockedError do
      # Can't lock it as a different user either
      job.lock current_user.uuid
    end
    job.reload
    assert_equal "Running", job.state

    assert_raises ArvadosModel::PermissionDeniedError do
      # Can't update fields as a different user
      job.update_attributes(state: "Failed")
    end
    job.reload
    assert_equal "Running", job.state


    set_user_from_auth :active_trustedclient

    # Can update fields as the locked_by user
    job.update_attributes(state: "Failed")
    assert_equal "Failed", job.state
  end

  test "admin user can cancel a running job despite lock" do
    set_user_from_auth :active_trustedclient
    job = Job.create! job_attrs
    job.lock current_user.uuid
    assert_equal Job::Running, job.state

    set_user_from_auth :spectator
    assert_raises do
      job.update_attributes!(state: Job::Cancelled)
    end

    set_user_from_auth :admin
    job.reload
    assert_equal Job::Running, job.state
    job.update_attributes!(state: Job::Cancelled)
    assert_equal Job::Cancelled, job.state
  end

  test "verify job queue position" do
    job1 = Job.create! job_attrs
    assert_equal 'Queued', job1.state, "Incorrect job state for newly created job1"

    job2 = Job.create! job_attrs
    assert_equal 'Queued', job2.state, "Incorrect job state for newly created job2"

    assert_not_nil job1.queue_position, "Expected non-nil queue position for job1"
    assert_not_nil job2.queue_position, "Expected non-nil queue position for job2"
  end

  SDK_MASTER = "ca68b24e51992e790f29df5cc4bc54ce1da4a1c2"
  SDK_TAGGED = "00634b2b8a492d6f121e3cf1d6587b821136a9a7"

  def sdk_constraint(version)
    {runtime_constraints: {
        "arvados_sdk_version" => version,
        "docker_image" => links(:docker_image_collection_tag).name,
      }}
  end

  def check_job_sdk_version(expected)
    job = yield
    if expected.nil?
      refute(job.valid?, "job valid with bad Arvados SDK version")
    else
      assert(job.valid?, "job not valid with good Arvados SDK version")
      assert_equal(expected, job.arvados_sdk_version)
    end
  end

  { "master" => SDK_MASTER,
    "commit2" => SDK_TAGGED,
    SDK_TAGGED[0, 8] => SDK_TAGGED,
    "__nonexistent__" => nil,
  }.each_pair do |search, commit_hash|
    test "creating job with SDK version '#{search}'" do
      check_job_sdk_version(commit_hash) do
        Job.new(job_attrs(sdk_constraint(search)))
      end
    end

    test "updating job from no SDK to version '#{search}'" do
      job = Job.create!(job_attrs)
      assert_nil job.arvados_sdk_version
      check_job_sdk_version(commit_hash) do
        job.runtime_constraints = sdk_constraint(search)[:runtime_constraints]
        job
      end
    end

    test "updating job from SDK version 'master' to '#{search}'" do
      job = Job.create!(job_attrs(sdk_constraint("master")))
      assert_equal(SDK_MASTER, job.arvados_sdk_version)
      check_job_sdk_version(commit_hash) do
        job.runtime_constraints = sdk_constraint(search)[:runtime_constraints]
        job
      end
    end
  end

  test "clear the SDK version" do
    job = Job.create!(job_attrs(sdk_constraint("master")))
    assert_equal(SDK_MASTER, job.arvados_sdk_version)
    job.runtime_constraints = {}
    assert(job.valid?, "job invalid after clearing SDK version")
    assert_nil(job.arvados_sdk_version)
  end

  test "job with SDK constraint, without Docker image is invalid" do
    sdk_attrs = sdk_constraint("master")
    sdk_attrs[:runtime_constraints].delete("docker_image")
    job = Job.create(job_attrs(sdk_attrs))
    refute(job.valid?, "Job valid with SDK version, without Docker image")
    sdk_errors = job.errors.messages[:arvados_sdk_version] || []
    refute_empty(sdk_errors.grep(/\bDocker\b/),
                 "no Job SDK errors mention that Docker is required")
  end

  test "invalid to clear Docker image constraint when SDK constraint exists" do
    job = Job.create!(job_attrs(sdk_constraint("master")))
    job.runtime_constraints.delete("docker_image")
    refute(job.valid?,
           "Job with SDK constraint valid after clearing Docker image")
  end

  test "use migrated docker image if requesting old-format image by tag" do
    Rails.configuration.docker_image_formats = ['v2']
    add_docker19_migration_link
    job = Job.create!(
      job_attrs(
        script: 'foo',
        runtime_constraints: {
          'docker_image' => links(:docker_image_collection_tag).name}))
    assert(job.valid?)
    assert_equal(job.docker_image_locator, collections(:docker_image_1_12).portable_data_hash)
  end

  test "use migrated docker image if requesting old-format image by pdh" do
    Rails.configuration.docker_image_formats = ['v2']
    add_docker19_migration_link
    job = Job.create!(
      job_attrs(
        script: 'foo',
        runtime_constraints: {
          'docker_image' => collections(:docker_image).portable_data_hash}))
    assert(job.valid?)
    assert_equal(job.docker_image_locator, collections(:docker_image_1_12).portable_data_hash)
  end

  [[:docker_image, :docker_image, :docker_image_1_12],
   [:docker_image_1_12, :docker_image, :docker_image_1_12],
   [:docker_image, :docker_image_1_12, :docker_image_1_12],
   [:docker_image_1_12, :docker_image_1_12, :docker_image_1_12],
  ].each do |existing_image, request_image, expect_image|
    test "if a #{existing_image} job exists, #{request_image} yields #{expect_image} after migration" do
      Rails.configuration.docker_image_formats = ['v1']

      if existing_image == :docker_image
        oldjob = Job.create!(
          job_attrs(
            script: 'foobar1',
            runtime_constraints: {
              'docker_image' => collections(existing_image).portable_data_hash}))
        oldjob.reload
        assert_equal(oldjob.docker_image_locator,
                     collections(existing_image).portable_data_hash)
      elsif existing_image == :docker_image_1_12
        assert_raises(ActiveRecord::RecordInvalid,
                      "Should not resolve v2 image when only v1 is supported") do
        oldjob = Job.create!(
          job_attrs(
            script: 'foobar1',
            runtime_constraints: {
              'docker_image' => collections(existing_image).portable_data_hash}))
        end
      end

      Rails.configuration.docker_image_formats = ['v2']
      add_docker19_migration_link

      # Check that both v1 and v2 images get resolved to v2.
      newjob = Job.create!(
        job_attrs(
          script: 'foobar1',
          runtime_constraints: {
            'docker_image' => collections(request_image).portable_data_hash}))
      newjob.reload
      assert_equal(newjob.docker_image_locator,
                   collections(expect_image).portable_data_hash)
    end
  end

  test "can't create job with SDK version assigned directly" do
    check_creation_prohibited(arvados_sdk_version: SDK_MASTER)
  end

  test "can't modify job to assign SDK version directly" do
    check_modification_prohibited(arvados_sdk_version: SDK_MASTER)
  end

  test "job validation fails when collection uuid found in script_parameters" do
    bad_params = {
      script_parameters: {
        'input' => {
          'param1' => 'the collection uuid zzzzz-4zz18-012345678901234'
        }
      }
    }
    assert_raises(ActiveRecord::RecordInvalid,
                  "created job with a collection uuid in script_parameters") do
      Job.create!(job_attrs(bad_params))
    end
  end

  test "job validation succeeds when no collection uuid in script_parameters" do
    good_params = {
      script_parameters: {
        'arg1' => 'foo',
        'arg2' => [ 'bar', 'baz' ],
        'arg3' => {
          'a' => 1,
          'b' => [2, 3, 4],
        }
      }
    }
    job = Job.create!(job_attrs(good_params))
    assert job.valid?
  end

  test 'update job uuid tag in internal.git when version changes' do
    authorize_with :active
    j = jobs :queued
    j.update_attributes repository: 'active/foo', script_version: 'b1'
    assert_equal('1de84a854e2b440dc53bf42f8548afa4c17da332',
                 internal_tag(j.uuid))
    j.update_attributes repository: 'active/foo', script_version: 'master'
    assert_equal('077ba2ad3ea24a929091a9e6ce545c93199b8e57',
                 internal_tag(j.uuid))
  end

  test 'script_parameters_digest is independent of key order' do
    j1 = Job.new(job_attrs(script_parameters: {'a' => 'a', 'ddee' => {'d' => 'd', 'e' => 'e'}}))
    j2 = Job.new(job_attrs(script_parameters: {'ddee' => {'e' => 'e', 'd' => 'd'}, 'a' => 'a'}))
    assert j1.valid?
    assert j2.valid?
    assert_equal(j1.script_parameters_digest, j2.script_parameters_digest)
  end

  test 'job fixtures have correct script_parameters_digest' do
    Job.all.each do |j|
      d = j.script_parameters_digest
      assert_equal(j.update_script_parameters_digest, d,
                   "wrong script_parameters_digest for #{j.uuid}")
    end
  end

  test 'deep_sort_hash on array of hashes' do
    a = {'z' => [[{'a' => 'a', 'b' => 'b'}]]}
    b = {'z' => [[{'b' => 'b', 'a' => 'a'}]]}
    assert_equal Job.deep_sort_hash(a).to_json, Job.deep_sort_hash(b).to_json
  end

  test 'find_reusable without logging' do
    Rails.logger.expects(:info).never
    try_find_reusable
  end

  test 'find_reusable with logging' do
    Rails.configuration.log_reuse_decisions = true
    Rails.logger.expects(:info).at_least(3)
    try_find_reusable
  end

  def try_find_reusable
    foobar = jobs(:foobar)
    example_attrs = {
      script_version: foobar.script_version,
      script: foobar.script,
      script_parameters: foobar.script_parameters,
      repository: foobar.repository,
    }

    # Two matching jobs exist with identical outputs. The older one
    # should be reused.
    j = Job.find_reusable(example_attrs, {}, [], [users(:active)])
    assert j
    assert_equal foobar.uuid, j.uuid

    # Two matching jobs exist with different outputs. Neither should
    # be reused.
    Job.where(uuid: jobs(:job_with_latest_version).uuid).
      update_all(output: 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa+1')
    assert_nil Job.find_reusable(example_attrs, {}, [], [users(:active)])

    # ...unless config says to reuse the earlier job in such cases.
    Rails.configuration.reuse_job_if_outputs_differ = true
    j = Job.find_reusable(example_attrs, {}, [], [users(:active)])
    assert_equal foobar.uuid, j.uuid
  end

  [
    true,
    false,
  ].each do |cascade|
    test "cancel job with cascade #{cascade}" do
      job = Job.find_by_uuid jobs(:running_job_with_components_at_level_1).uuid
      job.cancel cascade: cascade
      assert_equal Job::Cancelled, job.state

      descendents = ['zzzzz-8i9sb-jobcomponentsl2',
                     'zzzzz-d1hrv-picomponentsl02',
                     'zzzzz-8i9sb-job1atlevel3noc',
                     'zzzzz-8i9sb-job2atlevel3noc']

      jobs = Job.where(uuid: descendents)
      jobs.each do |j|
        assert_equal ('Cancelled' == j.state), cascade
      end

      pipelines = PipelineInstance.where(uuid: descendents)
      pipelines.each do |pi|
        assert_equal ('Paused' == pi.state), cascade
      end
    end
  end

  test 'cancelling a completed job raises error' do
    job = Job.find_by_uuid jobs(:job_with_latest_version).uuid
    assert job
    assert_equal 'Complete', job.state

    assert_raises(ArvadosModel::InvalidStateTransitionError) do
      job.cancel
    end
  end

  test 'cancelling a job with circular relationship with another does not result in an infinite loop' do
    job = Job.find_by_uuid jobs(:running_job_2_with_circular_component_relationship).uuid

    job.cancel cascade: true

    assert_equal Job::Cancelled, job.state

    child = Job.find_by_uuid job.components.collect{|_, uuid| uuid}[0]
    assert_equal Job::Cancelled, child.state
  end
end
