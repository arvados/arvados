# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class Node < ArvadosBase
  def self.creatable?
    false
  end
  def friendly_link_name lookup=nil
    (hostname && !hostname.empty?) ? hostname : uuid
  end
end
