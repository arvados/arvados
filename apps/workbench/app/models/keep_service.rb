# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class KeepService < ArvadosBase
  def self.creatable?
    false
  end
end
