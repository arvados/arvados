# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

require "arvados"
require "mocha/minitest"

class FakeError < StandardError; end
class RequestIdTest < Minitest::Test
    def test_raise_exception_with_request_id
        arv = Arvados.new
        clnt = arv.client
        assert_nil clnt.request_id

        Google::APIClient.any_instance.stubs(:execute).raises(FakeError.new("Uh-oh..."))
        err = assert_raises(FakeError) do
            arv.collection.get(uuid: "zzzzz-4zz18-zzzzzzzzzzzzzzz")
        end
        assert clnt.request_id != nil
        assert_match /Uh-oh.*\(Request ID: req-[0-9a-zA-Z]{20}\)/, err.message
    end
end