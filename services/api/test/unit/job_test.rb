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
      script_version: "main",
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
      Rails.configuration.Containers.JobsAPI.DefaultDockerImage = default_docker_image if use_config

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
      Rails.configuration.RemoteClusters = ConfigLoader.to_OrderedOptions({})
      job = Job.new job_attrs(runtime_constraints:
                              {'docker_image' => image_spec})
      assert(job.invalid?, "nonexistent Docker image #{spec_type} #{image_spec} was valid")
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

  test "can't create job with SDK version assigned directly" do
    check_creation_prohibited(arvados_sdk_version: SDK_MASTER)
  end

  test "can't modify job to assign SDK version directly" do
    check_modification_prohibited(arvados_sdk_version: SDK_MASTER)
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
    Rails.configuration.Containers.JobsAPI.ReuseJobIfOutputsDiffer = true
    j = Job.find_reusable(example_attrs, {}, [], [users(:active)])
    assert_equal foobar.uuid, j.uuid
  end

  test 'enable legacy api configuration option = true' do
    Rails.configuration.Containers.JobsAPI.Enable = "true"
    check_enable_legacy_jobs_api
    assert_equal(Disable_update_jobs_api_method_list, Rails.configuration.API.DisabledAPIs)
  end

  test 'enable legacy api configuration option = false' do
    Rails.configuration.Containers.JobsAPI.Enable = "false"
    check_enable_legacy_jobs_api
    assert_equal Disable_jobs_api_method_list, Rails.configuration.API.DisabledAPIs
  end

  test 'enable legacy api configuration option = auto, has jobs' do
    Rails.configuration.Containers.JobsAPI.Enable = "auto"
    assert Job.count > 0
    check_enable_legacy_jobs_api
    assert_equal(Disable_update_jobs_api_method_list, Rails.configuration.API.DisabledAPIs)
  end

  test 'enable legacy api configuration option = auto, no jobs' do
    Rails.configuration.Containers.JobsAPI.Enable = "auto"
    act_as_system_user do
      Job.destroy_all
    end
    assert_equal 0, Job.count
    assert_equal({}, Rails.configuration.API.DisabledAPIs)
    check_enable_legacy_jobs_api
    assert_equal Disable_jobs_api_method_list, Rails.configuration.API.DisabledAPIs
  end
end
