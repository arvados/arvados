require 'test_helper'
load 'test/functional/arvados/v1/git_setup.rb'

class CrunchDispatchTest < ActionDispatch::IntegrationTest
  include GitSetup

  fixtures :all

  @@crunch_dispatch_pid = nil

  def launch_crunch_dispatch
    @@crunch_dispatch_pid = Process.fork {
      ENV['PATH'] = ENV['HOME'] + '/arvados/services/crunch:' + ENV['PATH']
      exec(ENV['HOME'] + '/arvados/services/api/script/crunch-dispatch.rb')
    }
  end

  teardown do
    if @@crunch_dispatch_pid
      Process.kill "TERM", @@crunch_dispatch_pid
      Process.wait
      @@crunch_dispatch_pid = nil
    end
  end

  test "job runs" do
    post "/arvados/v1/jobs", {
      format: "json",
      job: {
        script: "log",
        repository: "bar",
        script_version: "143fec09e988160673c63457fa12a0f70b5b8a26",
        script_parameters: "{}"
      }
    }, auth(:admin)
    assert_response :success
  end
end
