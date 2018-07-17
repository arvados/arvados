# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class SafeJSON
  def self.dump(o)
    return Oj.dump(o, mode: :compat)
  end
  def self.load(s)
    if s.nil? or s == ''
      # Oj 2.18.5 used to return nil. Not anymore on 3.6.4.
      # Upgraded for performance issues (see #13803 and
      # https://github.com/ohler55/oj/issues/441)
      return nil
    end
    Oj.strict_load(s, symbol_keys: false)
  end
end
