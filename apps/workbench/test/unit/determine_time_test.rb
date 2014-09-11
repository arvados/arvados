require 'test_helper'

class DetermineTimeTest < ActiveSupport::TestCase
  test "one" do
    r1 = [{started_at: 1, finished_at: 3}]
    assert_equal 2, determine_wallclock_runtime(r1)
  end
end
