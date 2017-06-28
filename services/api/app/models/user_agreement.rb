# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class UserAgreement < Collection
  # This class exists so that Arvados::V1::SchemaController includes
  # UserAgreementsController's methods in the discovery document.
end
