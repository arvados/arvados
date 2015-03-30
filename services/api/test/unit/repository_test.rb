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

  test 'write permission not sufficient for changing name' do
    act_as_user users(:active) do
      r = repositories(:foo)
      name_was = r.name
      r.name = 'newname'
      assert_raises ArvadosModel::PermissionDeniedError do
        r.save!
      end
      r.reload
      assert_equal name_was, r.name
    end
  end

  test 'write permission necessary for changing modified_at' do
    act_as_user users(:spectator) do
      r = repositories(:foo)
      modtime_was = r.modified_at
      r.modified_at = Time.now
      assert_raises ArvadosModel::PermissionDeniedError do
        r.save!
      end
      r.reload
      assert_equal modtime_was, r.modified_at
    end
  end
end
