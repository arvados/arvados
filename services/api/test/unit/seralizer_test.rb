# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'
require 'serializers'

class SerializerTest < ActiveSupport::TestCase
  test 'serialize' do
    assert_equal('{}', HashSerializer.dump({}))
    assert_equal('{"foo":"bar"}', HashSerializer.dump(foo: 'bar'))
    assert_equal('{"foo":"bar"}', HashSerializer.dump('foo' => 'bar'))
    assert_equal('[]', ArraySerializer.dump([]))
    assert_equal('["foo",{"foo":"bar"}]',
                 ArraySerializer.dump(['foo', 'foo' => 'bar']))
    assert_equal(['foo'],
                 ArraySerializer.load(ArraySerializer.dump([:foo])))
    assert_equal([1,'bar'],
                 ArraySerializer.load(ArraySerializer.dump([1,'bar'])))
  end

  test 'load array that was saved as json, then mangled by an old version' do
    assert_equal(['foo'],
                 ArraySerializer.load(YAML.dump(ArraySerializer.dump(['foo']))))
  end
end
