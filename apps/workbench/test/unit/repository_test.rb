require 'test_helper'

class RepositoryTest < ActiveSupport::TestCase
  [
    ['admin', true],
    ['active', false],
  ].each do |user, can_edit|
    test "#{user} can edit attributes #{can_edit}" do
      use_token user
      attrs = Repository.new.editable_attributes
      if can_edit
        refute_empty attrs
      else
        assert_empty attrs
      end
    end
  end
end
