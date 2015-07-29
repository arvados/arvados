require 'minitest/autorun'

class TestCrunchJob < Minitest::Test
  SPECIAL_EXIT = {
    EX_RETRY_UNLOCKED: 93,
    EX_TEMPFAIL: 75,
  }

  JOBSPEC = {
    grep_local: {
      script: 'grep',
      script_version: 'master',
      repository: File.absolute_path('../../../..', __FILE__),
      script_parameters: {foo: 'bar'},
    },
  }

  def setup
  end

  def crunchjob
    File.absolute_path '../../bin/crunch-job', __FILE__
  end

  # Return environment suitable for running crunch-job.
  def crunchenv opts={}
    env = ENV.to_h
    env['PERLLIB'] = File.absolute_path('../../../perl/lib', __FILE__)
    env
  end

  def jobspec label
    JOBSPEC[label].dup
  end

  # Encode job record to json and run it with crunch-job.
  #
  # opts[:binstubs] is an array of X where ./binstub_X is added to
  # PATH in order to mock system programs.
  def tryjobrecord jobrecord, opts={}
    env = crunchenv
    (opts[:binstubs] || []).each do |binstub|
      env['PATH'] = File.absolute_path('../binstub_'+binstub, __FILE__) + ':' + env['PATH']
    end
    system env, crunchjob, '--job', jobrecord.to_json
  end

  def test_bogus_json
    out, err = capture_subprocess_io do
      system crunchenv, crunchjob, '--job', '"}{"'
    end
    assert_equal false, $?.success?
    # Must not conflict with our special exit statuses
    assert_jobfail $?
    assert_match /JSON/, err
  end

  def test_fail_sanity_check
    out, err = capture_subprocess_io do
      j = {}
      tryjobrecord j, binstubs: ['sanity_check']
    end
    assert_equal 75, $?.exitstatus
    assert_match /Sanity check failed: 7/, err
  end

  def test_fail_docker_sanity_check
    out, err = capture_subprocess_io do
      j = {}
      j[:docker_image_locator] = '4d449b9d34f2e2222747ef79c53fa3ff+1234'
      tryjobrecord j, binstubs: ['sanity_check']
    end
    assert_equal 75, $?.exitstatus
    assert_match /Sanity check failed: 8/, err
  end

  def test_no_script_specified
    out, err = capture_subprocess_io do
      j = jobspec :grep_local
      j.delete :script
      tryjobrecord j
    end
    assert_match /No script specified/, err
    assert_jobfail $?
  end

  def test_fail_clean_tmp
    out, err = capture_subprocess_io do
      j = jobspec :grep_local
      tryjobrecord j, binstubs: ['clean_fail']
    end
    assert_match /Failing mount stub was called/, err
    assert_match /Clean work dirs: exit 1\n$/, err
    assert_equal SPECIAL_EXIT[:EX_RETRY_UNLOCKED], $?.exitstatus
  end

  def test_docker_image_missing
    skip 'API bug: it refuses to create this job in Running state'
    out, err = capture_subprocess_io do
      j = jobspec :grep_local
      j[:docker_image_locator] = '4d449b9d34f2e2222747ef79c53fa3ff+1234'
      tryjobrecord j, binstubs: ['docker_noop']
    end
    assert_match /No Docker image hash found from locator/, err
    assert_jobfail $?
  end

  def test_script_version_not_found_in_repository
    bogus_version = 'f8b72707c1f5f740dbf1ed56eb429a36e0dee770'
    out, err = capture_subprocess_io do
      j = jobspec :grep_local
      j[:script_version] = bogus_version
      tryjobrecord j
    end
    assert_match /'#{bogus_version}' not found, giving up/, err
    assert_jobfail $?
  end

  # Ensure procstatus is not interpreted as a temporary infrastructure
  # problem. Would be assert_http_4xx if this were http.
  def assert_jobfail procstatus
    refute_includes SPECIAL_EXIT.values, procstatus.exitstatus
    assert_equal false, procstatus.success?
  end
end
