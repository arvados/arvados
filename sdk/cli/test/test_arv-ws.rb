# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

require 'minitest/autorun'

class TestArvWs < Minitest::Test
  def setup
  end

  def test_arv_ws_get_help
    out, err = capture_subprocess_io do
      system ('arv-ws -h')
    end
    assert_equal '', err
  end

  def test_arv_ws_such_option
    out, err = capture_subprocess_io do
      system ('arv-ws --junk')
    end
    refute_equal '', err
  end

end
