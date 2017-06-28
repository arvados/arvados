# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class ApiClientAuthorization < ArvadosBase
  def editable_attributes
    %w(expires_at default_owner_uuid)
  end
  def self.creatable?
    false
  end
end
