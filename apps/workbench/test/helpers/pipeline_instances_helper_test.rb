require 'test_helper'

class PipelineInstancesHelperTest < ActionView::TestCase
  test "one" do
    r = [{started_at: 1, finished_at: 3}]
    assert_equal 2, determine_wallclock_runtime(r)

    r = [{started_at: 1, finished_at: 5}]
    assert_equal 4, determine_wallclock_runtime(r)

    r = [{started_at: 1, finished_at: 2}, {started_at: 3, finished_at: 5}]
    assert_equal 3, determine_wallclock_runtime(r)

    r = [{started_at: 3, finished_at: 5}, {started_at: 1, finished_at: 2}]
    assert_equal 3, determine_wallclock_runtime(r)

    r = [{started_at: 3, finished_at: 5}, {started_at: 1, finished_at: 2},
         {started_at: 2, finished_at: 4}]
    assert_equal 4, determine_wallclock_runtime(r)

    r = [{started_at: 1, finished_at: 5}, {started_at: 2, finished_at: 3}]
    assert_equal 4, determine_wallclock_runtime(r)

    r = [{started_at: 3, finished_at: 5}, {started_at: 1, finished_at: 4}]
    assert_equal 4, determine_wallclock_runtime(r)

    r = [{started_at: 1, finished_at: 4}, {started_at: 3, finished_at: 5}]
    assert_equal 4, determine_wallclock_runtime(r)

    r = [{started_at: 1, finished_at: 4}, {started_at: 3, finished_at: 5},
         {started_at: 5, finished_at: 8}]
    assert_equal 7, determine_wallclock_runtime(r)

    r = [{started_at: 1, finished_at: 4}, {started_at: 3, finished_at: 5},
         {started_at: 6, finished_at: 8}]
    assert_equal 6, determine_wallclock_runtime(r)
  end
end
