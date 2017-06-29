# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

module LogReuseInfo
  # log_reuse_info logs whatever the given block returns, if
  # log_reuse_decisions is enabled. It accepts a block instead of a
  # string because in some cases constructing the strings involves
  # doing expensive things like database queries, and we want to skip
  # those when logging is disabled.
  def log_reuse_info(candidates=nil)
    if Rails.configuration.log_reuse_decisions
      msg = yield
      if !candidates.nil?
        msg = "have #{candidates.count} candidates " + msg
      end
      Rails.logger.info("find_reusable: " + msg)
    end
  end
end
