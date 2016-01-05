require 'test_helper'
require 'crunch_dispatch'

class FailJobsTest < ActiveSupport::TestCase
  include DbCurrentTime

  BOOT_TIME = 1448378837

  setup do
    @job = {}
    act_as_user users(:admin) do
      @job[:before_reboot] = Job.create!(state: 'Running',
                                         running: true,
                                         started_at: Time.at(BOOT_TIME - 300))
      @job[:after_reboot] = Job.create!(state: 'Running',
                                        running: true,
                                        started_at: Time.at(BOOT_TIME + 300))
      @job[:complete] = Job.create!(state: 'Running',
                                    running: true,
                                    started_at: Time.at(BOOT_TIME - 300))
      @job[:complete].update_attributes(state: 'Complete')
      @job[:complete].update_attributes(finished_at: Time.at(BOOT_TIME + 100))
      @job[:queued] = jobs(:queued)

      @job.values.each do |job|
        # backdate timestamps
        Job.where(uuid: job.uuid).
          update_all(created_at: Time.at(BOOT_TIME - 330),
                     modified_at: (job.finished_at ||
                                   job.started_at ||
                                   Time.at(BOOT_TIME - 300)))
      end
    end
    @dispatch = CrunchDispatch.new
    @test_start_time = db_current_time
  end

  test 'cancel slurm jobs' do
    Rails.configuration.crunch_job_wrapper = :slurm_immediate
    Rails.configuration.crunch_job_user = 'foobar'
    fake_squeue = File.popen("echo 1234 #{@job[:before_reboot].uuid}")
    fake_scancel = File.popen("true")
    File.expects(:popen).
      with(['squeue', '-h', '-o', '%i %j']).
      returns(fake_squeue)
    File.expects(:popen).
      with(includes('sudo', '-u', 'foobar', 'scancel', '1234')).
      returns(fake_scancel)
    @dispatch.fail_jobs(before: Time.at(BOOT_TIME).to_s)
    assert_end_states
  end

  test 'use reboot time' do
    Rails.configuration.crunch_job_wrapper = nil
    @dispatch.expects(:open).once.with('/proc/stat').
      returns open(Rails.root.join('test/fixtures/files/proc_stat'))
    @dispatch.fail_jobs(before: 'reboot')
    assert_end_states
  end

  test 'command line help' do
    cmd = Rails.root.join('script/fail-jobs.rb').to_s
    assert_match /Options:.*--before=/m, File.popen([cmd, '--help']).read
  end

  protected

  def assert_end_states
    @job.values.map &:reload
    assert_equal 'Failed', @job[:before_reboot].state
    assert_equal false, @job[:before_reboot].running
    assert_equal false, @job[:before_reboot].success
    assert_operator @job[:before_reboot].finished_at, :>=, @test_start_time
    assert_operator @job[:before_reboot].finished_at, :<=, db_current_time
    assert_equal 'Running', @job[:after_reboot].state
    assert_equal 'Complete', @job[:complete].state
    assert_equal 'Queued', @job[:queued].state
  end
end
