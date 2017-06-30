# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class ActiveSupport::TimeWithZone
  def as_json *args
    strftime "%Y-%m-%dT%H:%M:%S.%NZ"
  end
end
