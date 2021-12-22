# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

module CustomRequestId
  def make_request_id(req_id)
    if !req_id || req_id.length < 1 || req_id.length > 1024
      # Client-supplied ID is either missing or too long to be
      # considered friendly.
      internal_request_id
    else
      req_id
    end
  end

  def internal_request_id
    "req-" + Random::DEFAULT.rand(2**128).to_s(36)[0..19]
  end
end

class ActionDispatch::RequestId
  # Instead of using the default UUID-like format for X-Request-Id headers,
  # use our own.
  prepend CustomRequestId
end