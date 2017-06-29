# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class Log < ArvadosBase
  attr_accessor :object
  def self.creatable?
    # Technically yes, but not worth offering: it will be empty, and
    # you won't be able to edit it.
    false
  end
end
