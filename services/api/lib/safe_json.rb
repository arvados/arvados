# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class SafeJSON
  def self.dump(o)
    return Oj.dump(o, mode: :compat)
  end
  def self.load(s)
    Oj.strict_load(s, symbol_keys: false)
  end
end
