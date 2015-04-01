require 'test_helper'
require 'helpers/git_test_helper'

class CrunchDispatchTest < ActionDispatch::IntegrationTest
  include GitTestHelper

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
        repository: "active/crunchdispatchtest",
        script_version: "f35f99b7d32bac257f5989df02b9f12ee1a9b0d6",
        script_parameters: "{}"
      }
    }, auth(:admin)
    assert_response :success
  end
end
