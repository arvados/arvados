require 'test_helper'

class RepositoryTest < ActiveSupport::TestCase
  test 'write permission allows changing modified_at' do
    act_as_user users(:active) do
      r = repositories(:foo)
      modtime_was = r.modified_at
      r.modified_at = Time.now
      assert r.save
      assert_operator modtime_was, :<, r.modified_at
    end
  end
end
