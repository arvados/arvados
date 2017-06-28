# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class CollectionsHelperTest < ActionView::TestCase
  test "file_path generates short names" do
    assert_equal('foo', CollectionsHelper.file_path(['.', 'foo', 0]),
                 "wrong result for filename in collection root")
    assert_equal('foo/bar', CollectionsHelper.file_path(['foo', 'bar', 0]),
                 "wrong result for filename in directory without leading .")
    assert_equal('foo/bar', CollectionsHelper.file_path(['./foo', 'bar', 0]),
                 "wrong result for filename in directory with leading .")
  end
end
