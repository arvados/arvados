# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AuthorizedKey < ArvadosBase
  def attribute_editable?(attr, ever=nil)
    if (attr.to_s == 'authorized_user_uuid') and (not ever)
      current_user.andand.is_admin
    else
      super
    end
  end

  def self.creatable?
    false
  end
end
