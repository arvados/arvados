require 'test_helper'

class JobTaskTest < ActiveSupport::TestCase
  test "new tasks get an assigned qsequence" do
    set_user_from_auth :active
    task = JobTask.create
    assert_not_nil task.qsequence
    assert_operator(task.qsequence, :>=, 0)
  end

  test "assigned qsequence is not overwritten" do
    set_user_from_auth :active
    task = JobTask.create!(qsequence: 99)
    assert_equal(99, task.qsequence)
  end
end
