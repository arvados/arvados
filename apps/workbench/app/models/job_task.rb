# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class JobTask < ArvadosBase
  def work_unit(label=nil)
    JobTaskWorkUnit.new(self, label, self.uuid)
  end
end
