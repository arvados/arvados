# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class Container < ArvadosBase
  def self.creatable?
    false
  end

  def work_unit(label=nil, child_objects=nil)
    ContainerWorkUnit.new(self, label, self.uuid, child_objects=child_objects)
  end
end
